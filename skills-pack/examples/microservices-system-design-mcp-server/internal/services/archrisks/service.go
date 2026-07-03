// Package archrisks implements the service-layer logic for the
// detect_architecture_risks MCP tool.
//
// The service takes a structured description of a microservices architecture
// (components, their criticality and state, data stores, deployment target,
// constraints, and non-functional requirements) and produces a structured
// risk report: named architecture risks with severity, likelihood, affected
// components, and a concrete mitigation, plus a risk-posture score, the set
// of decisions still missing, and recommended next steps.
//
// The logic is deterministic and rule-based. It applies architecture-review
// heuristics that catch the operational and resilience risks that show up
// most often in real designs without LLM reasoning:
//
//   - Single points of failure (high fan-in, not replicated)
//   - Cascading failure (critical service on a deep synchronous chain)
//   - Missing resilience controls on critical components
//   - Stateful components with no durable store
//   - Shared data stores (scaling and blast-radius coupling)
//   - Unencrypted sensitive data (compliance exposure)
//   - Operational ambiguity (deployment target / SLO unspecified)
//   - Compliance constraints not reflected in the data model
//
// This package has no external dependencies and no LLM calls. Tests are
// table-driven and run in a few milliseconds.
package archrisks

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for an architecture risk assessment.
type Input struct {
	SystemName                string      `json:"system_name"`
	Description               string      `json:"description,omitempty"`
	Services                  []Component `json:"services"`
	DataStores                []DataStore `json:"data_stores,omitempty"`
	DeploymentTarget          string      `json:"deployment_target,omitempty"` // aks | container_apps | app_service | functions | hybrid
	Constraints               []string    `json:"constraints,omitempty"`
	NonFunctionalRequirements NFR         `json:"non_functional_requirements,omitempty"`
}

// Component describes a single service in the architecture.
type Component struct {
	Name               string   `json:"name"`
	Criticality        string   `json:"criticality,omitempty"` // high | medium | low (default: medium)
	Stateful           bool     `json:"stateful,omitempty"`
	Replicated         bool     `json:"replicated,omitempty"` // runs with >1 instance / HA
	DependsOn          []string `json:"depends_on,omitempty"` // synchronous dependencies
	ConsumesEventsFrom []string `json:"consumes_events_from,omitempty"`
	DataStores         []string `json:"data_stores,omitempty"` // data store names this component reads/writes
	Resilience         []string `json:"resilience,omitempty"`  // e.g., retry, circuit_breaker, timeout, bulkhead
}

// DataStore describes a data store referenced by components.
type DataStore struct {
	Name           string `json:"name"`
	Kind           string `json:"kind,omitempty"`           // e.g., postgres, cosmos, redis, blob
	Encrypted      bool   `json:"encrypted,omitempty"`      // encryption at rest enabled
	Classification string `json:"classification,omitempty"` // e.g., pii, phi, pci, sensitive, public
}

// NFR captures non-functional requirements relevant to risk assessment.
type NFR struct {
	AvailabilityTarget string `json:"availability_target,omitempty"` // e.g., "99.9"
	LatencyP99Ms       int    `json:"latency_p99_ms,omitempty"`
	RTOMinutes         int    `json:"rto_minutes,omitempty"`
	RPOMinutes         int    `json:"rpo_minutes,omitempty"`
}

// Report is the structured output: the architecture risk report.
type Report struct {
	SystemName       string   `json:"system_name"`
	Risks            []Risk   `json:"risks"`
	MissingDecisions []string `json:"missing_decisions"`
	NextSteps        []string `json:"next_steps"`
	RiskPostureScore int      `json:"risk_posture_score"` // 0-100, higher is healthier
	RiskRating       string   `json:"risk_rating"`
	Summary          string   `json:"summary"`
}

// Risk names an architecture-level problem with severity, likelihood, the
// components involved, and a concrete mitigation.
type Risk struct {
	Severity           string   `json:"severity"`   // low | medium | high
	Likelihood         string   `json:"likelihood"` // low | medium | high
	Category           string   `json:"category"`
	Description        string   `json:"description"`
	ComponentsAffected []string `json:"components_affected"`
	Mitigation         string   `json:"mitigation"`
}

// Service is the architecture risk service.
type Service struct{}

// NewService constructs a Service.
func NewService() *Service { return &Service{} }

// Detect applies the rule set and returns a Report.
//
// Errors are returned only for inputs that fundamentally cannot be processed
// (empty system name, no components). Issues within the input are surfaced as
// risks, not errors — the tool's job is to assess and report, not to reject.
func (s *Service) Detect(in Input) (Report, error) {
	if err := validate(in); err != nil {
		return Report{}, err
	}

	fanIn := computeFanIn(in.Services)
	dataStoreUsers := computeDataStoreUsers(in.Services)
	dataStoreByName := make(map[string]DataStore, len(in.DataStores))
	for _, ds := range in.DataStores {
		dataStoreByName[ds.Name] = ds
	}

	risks := detectRisks(in, fanIn, dataStoreUsers, dataStoreByName)
	missing := missingDecisions(in)
	score := computeScore(risks)
	rating := ratingForScore(score)
	next := nextSteps(risks)
	summary := composeSummary(in.SystemName, len(in.Services), len(risks), score)

	return Report{
		SystemName:       in.SystemName,
		Risks:            risks,
		MissingDecisions: missing,
		NextSteps:        next,
		RiskPostureScore: score,
		RiskRating:       rating,
		Summary:          summary,
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
	for _, c := range in.Services {
		if strings.TrimSpace(c.Name) == "" {
			return errors.New("every service must have a non-empty name")
		}
		if seen[c.Name] {
			return fmt.Errorf("duplicate service name: %s", c.Name)
		}
		seen[c.Name] = true
	}
	return nil
}

// computeFanIn returns, for each component, how many other components
// synchronously depend on it.
func computeFanIn(services []Component) map[string]int {
	fanIn := make(map[string]int)
	for _, c := range services {
		for _, dep := range c.DependsOn {
			fanIn[dep]++
		}
	}
	return fanIn
}

// computeDataStoreUsers returns, for each data store name, the sorted list of
// components that read or write it.
func computeDataStoreUsers(services []Component) map[string][]string {
	users := make(map[string][]string)
	for _, c := range services {
		for _, ds := range c.DataStores {
			users[ds] = append(users[ds], c.Name)
		}
	}
	for k := range users {
		sort.Strings(users[k])
	}
	return users
}

func criticalityOf(c Component) string {
	switch strings.ToLower(strings.TrimSpace(c.Criticality)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func hasResilience(c Component) bool {
	for _, r := range c.Resilience {
		if strings.TrimSpace(r) != "" {
			return true
		}
	}
	return false
}

var complianceKeywords = []string{"gdpr", "hipaa", "pci", "pii", "phi", "compliance", "sox", "soc2"}
var sensitiveClassifications = map[string]bool{"pii": true, "phi": true, "pci": true, "sensitive": true}

func detectRisks(in Input, fanIn map[string]int, dataStoreUsers map[string][]string, dataStoreByName map[string]DataStore) []Risk {
	risks := []Risk{}

	for _, c := range in.Services {
		// Single point of failure: high fan-in and not replicated.
		if n := fanIn[c.Name]; n > 3 && !c.Replicated {
			likelihood := "medium"
			if n > 5 {
				likelihood = "high"
			}
			risks = append(risks, Risk{
				Severity:           "high",
				Likelihood:         likelihood,
				Category:           "single_point_of_failure",
				Description:        fmt.Sprintf("service %q is a synchronous dependency of %d components and is not marked replicated; its failure takes down every dependent path", c.Name, n),
				ComponentsAffected: []string{c.Name},
				Mitigation:         "Run the service with multiple replicas behind a load balancer, add health probes, and verify dependents degrade gracefully (timeouts, fallbacks) when it is unavailable.",
			})
		}

		// Cascading failure: critical service on a deep synchronous chain.
		if criticalityOf(c) == "high" && len(c.DependsOn) >= 3 {
			risks = append(risks, Risk{
				Severity:           "high",
				Likelihood:         "medium",
				Category:           "cascading_failure",
				Description:        fmt.Sprintf("high-criticality service %q synchronously depends on %d services; a slowdown in any of them propagates to this critical path", c.Name, len(c.DependsOn)),
				ComponentsAffected: append([]string{c.Name}, sortedCopy(c.DependsOn)...),
				Mitigation:         "Introduce timeouts and circuit breakers on each synchronous call, and convert non-essential dependencies to asynchronous events so the critical path can complete degraded.",
			})
		}

		// Missing resilience controls on a critical component with sync deps.
		if criticalityOf(c) == "high" && !hasResilience(c) {
			severity := "medium"
			if len(c.DependsOn) > 0 {
				severity = "high"
			}
			risks = append(risks, Risk{
				Severity:           severity,
				Likelihood:         "medium",
				Category:           "missing_resilience",
				Description:        fmt.Sprintf("high-criticality service %q declares no resilience controls (retry, circuit_breaker, timeout, bulkhead)", c.Name),
				ComponentsAffected: []string{c.Name},
				Mitigation:         "Add explicit timeout, retry-with-backoff, and circuit-breaker policies on outbound calls; isolate critical work with bulkheads so one slow dependency cannot exhaust shared resources.",
			})
		}

		// Stateful component with no durable store.
		if c.Stateful && len(c.DataStores) == 0 {
			risks = append(risks, Risk{
				Severity:           "high",
				Likelihood:         "high",
				Category:           "stateful_without_datastore",
				Description:        fmt.Sprintf("service %q is marked stateful but references no data store; in-process state is lost on every restart, scale-in, or node failure", c.Name),
				ComponentsAffected: []string{c.Name},
				Mitigation:         "Externalize state to a managed data store (or make the service stateless and push state to the client/cache). Stateful in-memory services cannot scale horizontally or survive restarts.",
			})
		}
	}

	// Shared data store: a store used by more than one component.
	// Iterate in sorted key order so output is deterministic.
	sharedKeys := make([]string, 0, len(dataStoreUsers))
	for ds := range dataStoreUsers {
		sharedKeys = append(sharedKeys, ds)
	}
	sort.Strings(sharedKeys)
	for _, ds := range sharedKeys {
		users := dataStoreUsers[ds]
		if len(users) > 1 {
			severity := "medium"
			likelihood := "medium"
			if len(users) > 2 {
				likelihood = "high"
			}
			risks = append(risks, Risk{
				Severity:           severity,
				Likelihood:         likelihood,
				Category:           "shared_data_store",
				Description:        fmt.Sprintf("data store %q is accessed by %d components; shared stores couple deployment, scaling, and failure domains and erode service ownership", ds, len(users)),
				ComponentsAffected: append([]string{}, users...),
				Mitigation:         "Give the data store a single owning service; other components access it through that service's API or via published events, not direct connections.",
			})
		}
	}

	// Unencrypted sensitive data.
	for _, ds := range in.DataStores {
		if sensitiveClassifications[strings.ToLower(strings.TrimSpace(ds.Classification))] && !ds.Encrypted {
			risks = append(risks, Risk{
				Severity:           "high",
				Likelihood:         "high",
				Category:           "unencrypted_sensitive_data",
				Description:        fmt.Sprintf("data store %q is classified %q but is not marked encrypted at rest", ds.Name, ds.Classification),
				ComponentsAffected: append([]string{}, dataStoreUsers[ds.Name]...),
				Mitigation:         "Enable encryption at rest (platform-managed or customer-managed keys) and enforce encryption in transit; confirm key rotation and access auditing meet the relevant compliance regime.",
			})
		}
	}

	// Operational ambiguity: deployment target unspecified.
	if strings.TrimSpace(in.DeploymentTarget) == "" && len(in.Services) >= 3 {
		severity := "low"
		if len(in.Services) >= 5 {
			severity = "medium"
		}
		risks = append(risks, Risk{
			Severity:           severity,
			Likelihood:         "medium",
			Category:           "deployment_target_unspecified",
			Description:        fmt.Sprintf("no deployment target is specified for a %d-service system; scaling, networking, and operational model are undefined", len(in.Services)),
			ComponentsAffected: []string{},
			Mitigation:         "Choose and record a deployment target (AKS, Container Apps, App Service, Functions, or hybrid). The choice drives autoscaling, networking, and the operational runbook.",
		})
	}

	// No availability target while the system has a high-criticality service.
	if strings.TrimSpace(in.NonFunctionalRequirements.AvailabilityTarget) == "" && hasHighCriticality(in.Services) {
		risks = append(risks, Risk{
			Severity:           "medium",
			Likelihood:         "medium",
			Category:           "no_availability_target",
			Description:        "the system contains a high-criticality service but no availability target (SLO) is stated; resilience investment cannot be sized or verified against an undefined target",
			ComponentsAffected: highCriticalityNames(in.Services),
			Mitigation:         "Define an availability SLO (and an error budget) for the critical paths. The SLO determines redundancy, multi-zone/region needs, and the RTO/RPO targets.",
		})
	}

	// Compliance constraint not reflected in the data model.
	if hasComplianceConstraint(in.Constraints) && !anyClassified(in.DataStores) {
		risks = append(risks, Risk{
			Severity:           "high",
			Likelihood:         "medium",
			Category:           "compliance_constraint_unaddressed",
			Description:        "a compliance constraint is declared but no data store carries a data classification; the constraint is not traceable to the data model",
			ComponentsAffected: []string{},
			Mitigation:         "Classify every data store (pii, phi, pci, sensitive, public), then map each compliance constraint to controls (encryption, residency, retention, access audit) on the classified stores.",
		})
	}

	sort.SliceStable(risks, func(i, j int) bool {
		if risks[i].Severity != risks[j].Severity {
			return rank(risks[i].Severity) > rank(risks[j].Severity)
		}
		if risks[i].Likelihood != risks[j].Likelihood {
			return rank(risks[i].Likelihood) > rank(risks[j].Likelihood)
		}
		return risks[i].Category < risks[j].Category
	})

	return risks
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

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func hasHighCriticality(services []Component) bool {
	for _, c := range services {
		if criticalityOf(c) == "high" {
			return true
		}
	}
	return false
}

func highCriticalityNames(services []Component) []string {
	names := []string{}
	for _, c := range services {
		if criticalityOf(c) == "high" {
			names = append(names, c.Name)
		}
	}
	sort.Strings(names)
	return names
}

func hasComplianceConstraint(constraints []string) bool {
	for _, c := range constraints {
		lc := strings.ToLower(c)
		for _, kw := range complianceKeywords {
			if strings.Contains(lc, kw) {
				return true
			}
		}
	}
	return false
}

func anyClassified(stores []DataStore) bool {
	for _, ds := range stores {
		if strings.TrimSpace(ds.Classification) != "" {
			return true
		}
	}
	return false
}

func missingDecisions(in Input) []string {
	missing := []string{}
	if strings.TrimSpace(in.DeploymentTarget) == "" {
		missing = append(missing, "deployment target not chosen")
	}
	if strings.TrimSpace(in.NonFunctionalRequirements.AvailabilityTarget) == "" {
		missing = append(missing, "availability SLO not set")
	}
	unclassified := []string{}
	for _, c := range in.Services {
		if strings.TrimSpace(c.Criticality) == "" {
			unclassified = append(unclassified, c.Name)
		}
	}
	if len(unclassified) > 0 {
		sort.Strings(unclassified)
		missing = append(missing, "service criticality not classified for: "+strings.Join(unclassified, ", "))
	}
	if in.NonFunctionalRequirements.RTOMinutes == 0 && in.NonFunctionalRequirements.RPOMinutes == 0 {
		missing = append(missing, "recovery objectives (RTO/RPO) not stated")
	}
	return missing
}

// nextSteps returns the deduplicated mitigations of the highest-ranked risks,
// in risk order, capped at five.
func nextSteps(risks []Risk) []string {
	steps := []string{}
	seen := make(map[string]bool)
	for _, r := range risks {
		if seen[r.Mitigation] {
			continue
		}
		seen[r.Mitigation] = true
		steps = append(steps, r.Mitigation)
		if len(steps) == 5 {
			break
		}
	}
	return steps
}

// computeScore starts at 100 and deducts per risk using a severity x
// likelihood matrix. Floor at 0. Higher score = healthier risk posture.
func computeScore(risks []Risk) int {
	score := 100
	for _, r := range risks {
		score -= deduction(r.Severity, r.Likelihood)
	}
	if score < 0 {
		score = 0
	}
	return score
}

func deduction(severity, likelihood string) int {
	matrix := map[string]map[string]int{
		"high":   {"high": 20, "medium": 15, "low": 10},
		"medium": {"high": 10, "medium": 8, "low": 5},
		"low":    {"high": 3, "medium": 2, "low": 1},
	}
	if row, ok := matrix[severity]; ok {
		if v, ok := row[likelihood]; ok {
			return v
		}
	}
	return 0
}

func ratingForScore(score int) string {
	switch {
	case score >= 90:
		return "Low risk; architecture is operationally sound with minor follow-ups"
	case score >= 75:
		return "Moderate risk; targeted hardening recommended before launch"
	case score >= 60:
		return "Elevated risk; material issues to resolve before implementation"
	case score >= 40:
		return "High risk; significant rework recommended before proceeding"
	default:
		return "Severe risk; the architecture needs rework before it is operable"
	}
}

func composeSummary(systemName string, serviceCount, riskCount, score int) string {
	return fmt.Sprintf(
		"Architecture risk report for %s: %d services analyzed, %d risks identified, risk-posture score %d/100.",
		systemName, serviceCount, riskCount, score,
	)
}
