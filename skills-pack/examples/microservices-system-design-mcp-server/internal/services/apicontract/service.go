// Package apicontract implements the service-layer logic for the
// generate_api_contract MCP tool.
//
// The service takes a structured description of services and their resources
// and deterministically generates an OpenAPI-shaped API contract: REST
// endpoints per resource operation, standard status codes and error
// responses, and per-service security. It also reports contract-quality
// findings (unsecured endpoints, missing versioning, unpaginated list
// endpoints, inconsistent base paths) and an overall contract-readiness score.
//
// The logic is deterministic and rule-based: the same input always produces
// the same contract. No LLM calls, no external dependencies. Tests are
// table-driven and run in a few milliseconds.
package apicontract

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for generating an API contract.
type Input struct {
	SystemName         string    `json:"system_name"`
	Description        string    `json:"description,omitempty"`
	APIStyle           string    `json:"api_style,omitempty"`           // rest (default) | grpc | graphql
	VersioningStrategy string    `json:"versioning_strategy,omitempty"` // uri | header | none
	Services           []Service `json:"services"`
}

// Service describes a service whose API contract should be generated.
type Service struct {
	Name               string     `json:"name"`
	BusinessCapability string     `json:"business_capability,omitempty"`
	BasePath           string     `json:"base_path,omitempty"` // e.g., "/orders"
	Auth               string     `json:"auth,omitempty"`      // none | api_key | oauth2 | mtls
	Resources          []Resource `json:"resources"`
}

// Resource is a REST resource exposed by a service.
type Resource struct {
	Name       string   `json:"name"`
	Operations []string `json:"operations"` // list | get | create | update | delete
	Paginated  bool     `json:"paginated,omitempty"`
	Versioned  bool     `json:"versioned,omitempty"`
}

// Contract is the structured output: the generated API contract.
type Contract struct {
	SystemName       string            `json:"system_name"`
	APIContracts     []ServiceContract `json:"api_contracts"`
	ContractFindings []Finding         `json:"contract_findings"`
	OpenAPISummary   OpenAPISummary    `json:"openapi_summary"`
	ContractScore    int               `json:"contract_score"` // 0-100
	ContractRating   string            `json:"contract_rating"`
	Summary          string            `json:"summary"`
}

// ServiceContract is the generated contract for a single service.
type ServiceContract struct {
	Service   string     `json:"service"`
	BasePath  string     `json:"base_path"`
	Security  string     `json:"security"`
	Endpoints []Endpoint `json:"endpoints"`
}

// Endpoint is a single generated operation.
type Endpoint struct {
	Method         string `json:"method"`
	Path           string `json:"path"`
	Summary        string `json:"summary"`
	SuccessStatus  int    `json:"success_status"`
	ErrorResponses []int  `json:"error_responses"`
}

// Finding is a contract-quality issue.
type Finding struct {
	Severity         string   `json:"severity"` // low | medium | high
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	ServicesAffected []string `json:"services_affected"`
	Recommendation   string   `json:"recommendation"`
}

// OpenAPISummary is the headline OpenAPI metadata.
type OpenAPISummary struct {
	OpenAPIVersion  string `json:"openapi_version"`
	TotalPaths      int    `json:"total_paths"`
	TotalOperations int    `json:"total_operations"`
}

// Service is the API-contract service.
type Generator struct{}

// NewService constructs a Generator.
func NewService() *Generator { return &Generator{} }

// Generate applies the rule set and returns a Contract.
//
// Errors are returned only for inputs that fundamentally cannot be processed
// (empty system name, no services). Quality issues within the input are
// surfaced as findings, not errors.
func (g *Generator) Generate(in Input) (Contract, error) {
	if err := validate(in); err != nil {
		return Contract{}, err
	}

	contracts := make([]ServiceContract, 0, len(in.Services))
	totalPaths := 0
	totalOps := 0
	for _, svc := range in.Services {
		sc := buildServiceContract(svc)
		contracts = append(contracts, sc)
		totalPaths += countDistinctPaths(sc.Endpoints)
		totalOps += len(sc.Endpoints)
	}

	findings := detectFindings(in)
	score := computeScore(findings)
	rating := ratingForScore(score)

	return Contract{
		SystemName:       in.SystemName,
		APIContracts:     contracts,
		ContractFindings: findings,
		OpenAPISummary: OpenAPISummary{
			OpenAPIVersion:  "3.1.0",
			TotalPaths:      totalPaths,
			TotalOperations: totalOps,
		},
		ContractScore:  score,
		ContractRating: rating,
		Summary:        composeSummary(in.SystemName, len(in.Services), totalOps, len(findings), score),
	}, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Services) == 0 {
		return errors.New("at least one service is required")
	}
	seen := make(map[string]bool, len(in.Services))
	for _, svc := range in.Services {
		if strings.TrimSpace(svc.Name) == "" {
			return errors.New("every service must have a non-empty name")
		}
		if seen[svc.Name] {
			return fmt.Errorf("duplicate service name: %s", svc.Name)
		}
		seen[svc.Name] = true
	}
	return nil
}

func basePathFor(svc Service) string {
	bp := strings.TrimSpace(svc.BasePath)
	if bp == "" {
		bp = "/" + svc.Name
	}
	if !strings.HasPrefix(bp, "/") {
		bp = "/" + bp
	}
	return strings.TrimRight(bp, "/")
}

func securityFor(svc Service) string {
	switch strings.ToLower(strings.TrimSpace(svc.Auth)) {
	case "api_key":
		return "api_key"
	case "oauth2":
		return "oauth2"
	case "mtls":
		return "mtls"
	default:
		return "none"
	}
}

func buildServiceContract(svc Service) ServiceContract {
	base := basePathFor(svc)
	endpoints := []Endpoint{}
	for _, res := range svc.Resources {
		coll := fmt.Sprintf("%s/%s", base, res.Name)
		item := fmt.Sprintf("%s/%s/{id}", base, res.Name)
		for _, op := range res.Operations {
			switch strings.ToLower(strings.TrimSpace(op)) {
			case "list":
				endpoints = append(endpoints, Endpoint{
					Method: "GET", Path: coll,
					Summary:       fmt.Sprintf("List %s", res.Name),
					SuccessStatus: 200, ErrorResponses: []int{401, 500},
				})
			case "get":
				endpoints = append(endpoints, Endpoint{
					Method: "GET", Path: item,
					Summary:       fmt.Sprintf("Get a %s by id", res.Name),
					SuccessStatus: 200, ErrorResponses: []int{401, 404, 500},
				})
			case "create":
				endpoints = append(endpoints, Endpoint{
					Method: "POST", Path: coll,
					Summary:       fmt.Sprintf("Create a %s", res.Name),
					SuccessStatus: 201, ErrorResponses: []int{400, 401, 500},
				})
			case "update":
				endpoints = append(endpoints, Endpoint{
					Method: "PUT", Path: item,
					Summary:       fmt.Sprintf("Update a %s", res.Name),
					SuccessStatus: 200, ErrorResponses: []int{400, 401, 404, 500},
				})
			case "delete":
				endpoints = append(endpoints, Endpoint{
					Method: "DELETE", Path: item,
					Summary:       fmt.Sprintf("Delete a %s", res.Name),
					SuccessStatus: 204, ErrorResponses: []int{401, 404, 500},
				})
			}
		}
	}
	sort.SliceStable(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})
	return ServiceContract{
		Service:   svc.Name,
		BasePath:  base,
		Security:  securityFor(svc),
		Endpoints: endpoints,
	}
}

func countDistinctPaths(eps []Endpoint) int {
	seen := map[string]bool{}
	for _, e := range eps {
		seen[e.Path] = true
	}
	return len(seen)
}

func detectFindings(in Input) []Finding {
	findings := []Finding{}

	versioningGlobal := strings.ToLower(strings.TrimSpace(in.VersioningStrategy))
	if versioningGlobal == "" || versioningGlobal == "none" {
		findings = append(findings, Finding{
			Severity:         "medium",
			Category:         "missing_versioning",
			Description:      "no API versioning strategy is declared; breaking changes cannot be rolled out without disrupting clients",
			ServicesAffected: serviceNames(in.Services),
			Recommendation:   "Adopt a versioning strategy (URI prefix like /v1 or a version header) before the first external consumer integrates.",
		})
	}

	for _, svc := range in.Services {
		if securityFor(svc) == "none" {
			findings = append(findings, Finding{
				Severity:         "high",
				Category:         "unsecured_endpoint",
				Description:      fmt.Sprintf("service %q declares no authentication; all generated endpoints are unsecured", svc.Name),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Set auth to oauth2 (or mtls for service-to-service). Public read-only endpoints should still be explicit, not default.",
			})
		}
		if strings.TrimSpace(svc.BusinessCapability) == "" {
			findings = append(findings, Finding{
				Severity:         "low",
				Category:         "missing_capability",
				Description:      fmt.Sprintf("service %q has no declared business capability; API ownership and scope are ambiguous", svc.Name),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Declare the single business capability this API serves so the contract scope is unambiguous.",
			})
		}
		if bp := strings.TrimSpace(svc.BasePath); bp != "" && !strings.HasPrefix(bp, "/") {
			findings = append(findings, Finding{
				Severity:         "low",
				Category:         "inconsistent_base_path",
				Description:      fmt.Sprintf("service %q base_path %q does not start with '/'; path composition is inconsistent", svc.Name, bp),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Normalize base paths to begin with '/' and omit trailing slashes.",
			})
		}
		for _, res := range svc.Resources {
			if hasOp(res.Operations, "list") && !res.Paginated {
				findings = append(findings, Finding{
					Severity:         "medium",
					Category:         "no_pagination",
					Description:      fmt.Sprintf("resource %q in service %q exposes a list operation without pagination; unbounded responses degrade and eventually fail at scale", res.Name, svc.Name),
					ServicesAffected: []string{svc.Name},
					Recommendation:   "Add cursor or page/limit pagination to all list endpoints and document the default and maximum page size.",
				})
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return rank(findings[i].Severity) > rank(findings[j].Severity)
		}
		return findings[i].Category < findings[j].Category
	})
	return findings
}

func hasOp(ops []string, want string) bool {
	for _, o := range ops {
		if strings.EqualFold(strings.TrimSpace(o), want) {
			return true
		}
	}
	return false
}

func serviceNames(services []Service) []string {
	names := make([]string, 0, len(services))
	for _, s := range services {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

func rank(level string) int {
	switch level {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func computeScore(findings []Finding) int {
	score := 100
	for _, f := range findings {
		switch f.Severity {
		case "high":
			score -= 15
		case "medium":
			score -= 8
		case "low":
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

func ratingForScore(score int) string {
	switch {
	case score >= 90:
		return "Contract is production-ready; minor refinements only"
	case score >= 75:
		return "Contract is directionally sound; targeted fixes recommended"
	case score >= 60:
		return "Contract has material gaps; address before publishing"
	case score >= 40:
		return "Contract needs significant work before external consumers integrate"
	default:
		return "Contract is not ready; rework the API design"
	}
}

func composeSummary(systemName string, serviceCount, opCount, findingCount, score int) string {
	return fmt.Sprintf(
		"API contract for %s: %d services, %d operations generated, %d contract findings, contract-readiness score %d/100.",
		systemName, serviceCount, opCount, findingCount, score,
	)
}
