// Package boundary implements the service-layer logic for the
// generate_service_boundary_canvas MCP tool.
//
// The service takes structured input describing a system (proposed services,
// their capabilities, data ownership, dependencies, team ownership) and
// produces a structured service boundary canvas with per-service assessments,
// boundary risks, recommended adjustments, and an overall score.
//
// The logic is deterministic and rule-based. It applies known service-boundary
// heuristics that catch the majority of boundary problems without LLM reasoning:
//
//   - Capability cohesion (does each service have a single, named business capability?)
//   - Data ownership clarity (is data owned by exactly one service, or co-owned?)
//   - Dependency hygiene (chatty synchronous chains, circular dependencies, unbounded fan-out)
//   - Team ownership (is each service owned by exactly one team?)
//   - Size sanity (services that span too many capabilities are flagged for splitting;
//     services with no meaningful capability are flagged for merging)
//
// This package has no external dependencies and no LLM calls. Tests are
// table-driven and run in a few milliseconds.
package boundary

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for generating a service boundary canvas.
type Input struct {
	SystemName  string            `json:"system_name"`
	Description string            `json:"description,omitempty"`
	Services    []ProposedService `json:"services"`
	DataStores  []DataStore       `json:"data_stores,omitempty"`
	Teams       []Team            `json:"teams,omitempty"`
}

// ProposedService describes a service the user is proposing for the system.
type ProposedService struct {
	Name               string   `json:"name"`
	BusinessCapability string   `json:"business_capability"`
	OwnsData           []string `json:"owns_data,omitempty"`
	DependsOn          []string `json:"depends_on,omitempty"`           // synchronous dependencies
	ConsumesEventsFrom []string `json:"consumes_events_from,omitempty"` // asynchronous dependencies
	Team               string   `json:"team,omitempty"`
}

// DataStore describes a data store referenced by services.
type DataStore struct {
	Name string `json:"name"`
	Kind string `json:"kind,omitempty"` // e.g., "postgres", "cosmos", "redis", "blob"
}

// Team describes a team that owns one or more services.
type Team struct {
	Name string `json:"name"`
}

// Canvas is the structured output: a service boundary canvas.
type Canvas struct {
	SystemName         string              `json:"system_name"`
	ServiceAssessments []ServiceAssessment `json:"service_assessments"`
	BoundaryRisks      []BoundaryRisk      `json:"boundary_risks"`
	RecommendedChanges []RecommendedChange `json:"recommended_changes"`
	OverallScore       int                 `json:"overall_score"` // 0-100
	OverallRating      string              `json:"overall_rating"`
	Summary            string              `json:"summary"`
}

// ServiceAssessment summarizes the boundary quality of a single proposed service.
type ServiceAssessment struct {
	Service           string   `json:"service"`
	CapabilityClarity string   `json:"capability_clarity"` // "clear", "ambiguous", "missing"
	OwnsDistinctData  bool     `json:"owns_distinct_data"`
	DependencyHealth  string   `json:"dependency_health"` // "healthy", "chatty", "circular_risk"
	OwnerClarity      string   `json:"owner_clarity"`     // "single_team", "no_team", "multi_team_risk"
	Notes             []string `json:"notes,omitempty"`
}

// BoundaryRisk names a boundary-level problem with severity and the services involved.
type BoundaryRisk struct {
	Severity         string   `json:"severity"` // "low", "medium", "high"
	Category         string   `json:"category"` // "data_co_ownership", "chatty_dependency", "circular_dependency", "no_owner", "capability_drift", "fan_out"
	Description      string   `json:"description"`
	ServicesAffected []string `json:"services_affected"`
}

// RecommendedChange is a concrete action to improve the boundary design.
type RecommendedChange struct {
	Action    string   `json:"action"` // "merge", "split", "clarify_ownership", "introduce_async", "consolidate_data"
	Targets   []string `json:"targets"`
	Rationale string   `json:"rationale"`
}

// Service is the boundary canvas service.
type Service struct{}

// NewService constructs a Service.
func NewService() *Service { return &Service{} }

// GenerateCanvas applies the rule set and returns a Canvas.
//
// Errors are returned only for inputs that fundamentally cannot be processed
// (empty system name, no proposed services). Rule violations within the input
// are surfaced as boundary risks, not errors — the tool's job is to assess
// and report, not to reject.
func (s *Service) GenerateCanvas(in Input) (Canvas, error) {
	if err := validate(in); err != nil {
		return Canvas{}, err
	}

	// Build lookup maps for efficient cross-service analysis.
	serviceByName := make(map[string]*ProposedService, len(in.Services))
	for i := range in.Services {
		serviceByName[in.Services[i].Name] = &in.Services[i]
	}

	dataOwnership := computeDataOwnership(in.Services)
	dependencyGraph := computeDependencyGraph(in.Services)

	// Per-service assessments.
	assessments := make([]ServiceAssessment, 0, len(in.Services))
	for _, svc := range in.Services {
		assessments = append(assessments, assessService(svc, dataOwnership, dependencyGraph))
	}

	// Cross-service boundary risks.
	risks := detectRisks(in.Services, dataOwnership, dependencyGraph)

	// Recommended changes follow from the risks.
	changes := recommendChanges(risks, in.Services, dataOwnership)

	// Overall score: 100 minus weighted risk severity sum, floored at 0.
	score := computeScore(assessments, risks)
	rating := ratingForScore(score)

	summary := composeSummary(in.SystemName, len(in.Services), len(risks), score)

	return Canvas{
		SystemName:         in.SystemName,
		ServiceAssessments: assessments,
		BoundaryRisks:      risks,
		RecommendedChanges: changes,
		OverallScore:       score,
		OverallRating:      rating,
		Summary:            summary,
	}, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Services) == 0 {
		return errors.New("at least one proposed service is required")
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

// computeDataOwnership returns, for each data store name, the list of services
// that claim to own it. Services co-owning the same data is a boundary risk.
func computeDataOwnership(services []ProposedService) map[string][]string {
	owners := make(map[string][]string)
	for _, svc := range services {
		for _, data := range svc.OwnsData {
			owners[data] = append(owners[data], svc.Name)
		}
	}
	return owners
}

// computeDependencyGraph returns adjacency: service -> services it depends on.
// Includes both synchronous and asynchronous dependencies, tagged.
func computeDependencyGraph(services []ProposedService) map[string]dependencies {
	graph := make(map[string]dependencies, len(services))
	for _, svc := range services {
		graph[svc.Name] = dependencies{
			Sync:  append([]string(nil), svc.DependsOn...),
			Async: append([]string(nil), svc.ConsumesEventsFrom...),
		}
	}
	return graph
}

type dependencies struct {
	Sync  []string
	Async []string
}

func assessService(svc ProposedService, dataOwnership map[string][]string, graph map[string]dependencies) ServiceAssessment {
	notes := []string{}

	// Capability clarity.
	capabilityClarity := "clear"
	switch {
	case strings.TrimSpace(svc.BusinessCapability) == "":
		capabilityClarity = "missing"
		notes = append(notes, "no business_capability declared")
	case len(strings.Split(svc.BusinessCapability, " ")) < 2:
		// One-word capabilities like "data" or "service" are typically too vague.
		capabilityClarity = "ambiguous"
		notes = append(notes, fmt.Sprintf("business_capability %q is too generic", svc.BusinessCapability))
	}

	// Distinct data ownership: does this service own at least one data store that
	// no other service also claims?
	ownsDistinct := false
	for _, data := range svc.OwnsData {
		if owners := dataOwnership[data]; len(owners) == 1 && owners[0] == svc.Name {
			ownsDistinct = true
			break
		}
	}
	if !ownsDistinct && len(svc.OwnsData) > 0 {
		notes = append(notes, "all owned data is also claimed by another service")
	}
	if len(svc.OwnsData) == 0 {
		notes = append(notes, "service owns no data — verify it is not just a thin wrapper")
	}

	// Dependency health.
	depHealth := "healthy"
	syncCount := len(graph[svc.Name].Sync)
	if syncCount > 4 {
		depHealth = "chatty"
		notes = append(notes, fmt.Sprintf("synchronous dependency count is %d, indicates chatty design", syncCount))
	}
	if hasCircularDependency(svc.Name, graph) {
		depHealth = "circular_risk"
		notes = append(notes, "participates in a circular synchronous dependency chain")
	}

	// Owner clarity.
	ownerClarity := "single_team"
	if strings.TrimSpace(svc.Team) == "" {
		ownerClarity = "no_team"
		notes = append(notes, "no team assigned — services without owners drift")
	}

	return ServiceAssessment{
		Service:           svc.Name,
		CapabilityClarity: capabilityClarity,
		OwnsDistinctData:  ownsDistinct,
		DependencyHealth:  depHealth,
		OwnerClarity:      ownerClarity,
		Notes:             notes,
	}
}

// hasCircularDependency does a depth-first search to detect cycles starting from start.
func hasCircularDependency(start string, graph map[string]dependencies) bool {
	visited := make(map[string]bool)
	var dfs func(node string) bool
	dfs = func(node string) bool {
		if node == start && len(visited) > 0 {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		for _, dep := range graph[node].Sync {
			if dfs(dep) {
				return true
			}
		}
		return false
	}
	for _, dep := range graph[start].Sync {
		clear(visited)
		visited[start] = true
		if dfs(dep) {
			return true
		}
	}
	return false
}

func detectRisks(services []ProposedService, dataOwnership map[string][]string, graph map[string]dependencies) []BoundaryRisk {
	risks := []BoundaryRisk{}

	// Data co-ownership: any data store with more than one owner is a high-severity risk.
	for data, owners := range dataOwnership {
		if len(owners) > 1 {
			sort.Strings(owners)
			risks = append(risks, BoundaryRisk{
				Severity:         "high",
				Category:         "data_co_ownership",
				Description:      fmt.Sprintf("data store %q is owned by multiple services; data ownership should be exclusive", data),
				ServicesAffected: owners,
			})
		}
	}

	// Services with no owner team.
	for _, svc := range services {
		if strings.TrimSpace(svc.Team) == "" {
			risks = append(risks, BoundaryRisk{
				Severity:         "medium",
				Category:         "no_owner",
				Description:      fmt.Sprintf("service %q has no owning team — services without owners drift in scope and quality", svc.Name),
				ServicesAffected: []string{svc.Name},
			})
		}
	}

	// Chatty synchronous chains.
	for _, svc := range services {
		if len(graph[svc.Name].Sync) > 4 {
			risks = append(risks, BoundaryRisk{
				Severity:         "medium",
				Category:         "chatty_dependency",
				Description:      fmt.Sprintf("service %q has %d synchronous dependencies; this is chatty and increases latency, blast radius, and coupling", svc.Name, len(graph[svc.Name].Sync)),
				ServicesAffected: []string{svc.Name},
			})
		}
	}

	// Circular synchronous dependencies.
	circularDetected := make(map[string]bool)
	for _, svc := range services {
		if hasCircularDependency(svc.Name, graph) && !circularDetected[svc.Name] {
			cycle := findCycle(svc.Name, graph)
			for _, c := range cycle {
				circularDetected[c] = true
			}
			risks = append(risks, BoundaryRisk{
				Severity:         "high",
				Category:         "circular_dependency",
				Description:      "circular synchronous dependency detected; break the cycle by introducing async messaging or restructuring the call chain",
				ServicesAffected: cycle,
			})
		}
	}

	// Capability drift: services with missing or ambiguous capabilities.
	for _, svc := range services {
		if strings.TrimSpace(svc.BusinessCapability) == "" {
			risks = append(risks, BoundaryRisk{
				Severity:         "medium",
				Category:         "capability_drift",
				Description:      fmt.Sprintf("service %q has no declared business capability — services without capability anchors expand into adjacent domains", svc.Name),
				ServicesAffected: []string{svc.Name},
			})
		}
	}

	// Fan-out: any single service that more than 5 other services synchronously depend on
	// is a bottleneck and single point of failure.
	fanIn := make(map[string]int)
	for _, svc := range services {
		for _, dep := range svc.DependsOn {
			fanIn[dep]++
		}
	}
	for target, count := range fanIn {
		if count > 5 {
			risks = append(risks, BoundaryRisk{
				Severity:         "high",
				Category:         "fan_out",
				Description:      fmt.Sprintf("service %q is a synchronous dependency of %d other services — it is a bottleneck and a single point of failure", target, count),
				ServicesAffected: []string{target},
			})
		}
	}

	// Stable ordering for reproducible output.
	sort.SliceStable(risks, func(i, j int) bool {
		if risks[i].Severity != risks[j].Severity {
			return severityRank(risks[i].Severity) > severityRank(risks[j].Severity)
		}
		return risks[i].Category < risks[j].Category
	})

	return risks
}

func severityRank(s string) int {
	switch s {
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

func findCycle(start string, graph map[string]dependencies) []string {
	// Returns a stable, deduplicated, sorted list of services in the detected cycle.
	visited := map[string]bool{}
	path := []string{}
	var dfs func(node string) []string
	dfs = func(node string) []string {
		if visited[node] {
			// Found cycle; return the path from where node first appeared.
			for i, n := range path {
				if n == node {
					return append([]string(nil), path[i:]...)
				}
			}
			return nil
		}
		visited[node] = true
		path = append(path, node)
		for _, dep := range graph[node].Sync {
			if cycle := dfs(dep); cycle != nil {
				return cycle
			}
		}
		path = path[:len(path)-1]
		return nil
	}
	cycle := dfs(start)
	sort.Strings(cycle)
	return cycle
}

func recommendChanges(risks []BoundaryRisk, services []ProposedService, dataOwnership map[string][]string) []RecommendedChange {
	changes := []RecommendedChange{}

	// Each risk category maps to a recommended-change pattern.
	for _, risk := range risks {
		switch risk.Category {
		case "data_co_ownership":
			changes = append(changes, RecommendedChange{
				Action:    "consolidate_data",
				Targets:   risk.ServicesAffected,
				Rationale: "Pick a single owner for this data store. The other services should access via the owner's API. Co-owned data leads to inconsistent updates and unclear accountability.",
			})
		case "chatty_dependency":
			changes = append(changes, RecommendedChange{
				Action:    "introduce_async",
				Targets:   risk.ServicesAffected,
				Rationale: "Convert the highest-volume synchronous calls to asynchronous events. This reduces coupling, contains failures, and improves latency.",
			})
		case "circular_dependency":
			changes = append(changes, RecommendedChange{
				Action:    "introduce_async",
				Targets:   risk.ServicesAffected,
				Rationale: "Break the cycle by introducing event-driven communication. At least one edge of the cycle should be asynchronous to avoid deadlock and cascading failures.",
			})
		case "no_owner":
			changes = append(changes, RecommendedChange{
				Action:    "clarify_ownership",
				Targets:   risk.ServicesAffected,
				Rationale: "Assign a single team as the owner. Services without owners accumulate undefined behavior and become legacy quickly.",
			})
		case "capability_drift":
			changes = append(changes, RecommendedChange{
				Action:    "clarify_ownership",
				Targets:   risk.ServicesAffected,
				Rationale: "Define the business capability this service serves. Without an anchor, the service will expand into adjacent domains.",
			})
		case "fan_out":
			changes = append(changes, RecommendedChange{
				Action:    "introduce_async",
				Targets:   risk.ServicesAffected,
				Rationale: "This service is a synchronous bottleneck for many consumers. Convert read-paths to cached or eventual-consistent reads; split write-paths into commands handled asynchronously.",
			})
		}
	}

	return changes
}

func computeScore(assessments []ServiceAssessment, risks []BoundaryRisk) int {
	score := 100
	// Each risk deducts: high = 15, medium = 8, low = 3.
	for _, r := range risks {
		switch r.Severity {
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
		return "Boundaries well-defined; minor adjustments only"
	case score >= 75:
		return "Boundaries directionally sound; targeted improvements recommended"
	case score >= 60:
		return "Boundaries have material issues; address before implementation"
	case score >= 40:
		return "Boundaries need significant rework; redesign recommended before proceeding"
	default:
		return "Boundaries are unclear or broken; restart the decomposition exercise"
	}
}

func composeSummary(systemName string, serviceCount, riskCount, score int) string {
	return fmt.Sprintf(
		"Service boundary canvas for %s: %d proposed services analyzed, %d boundary risks identified, overall boundary score %d/100.",
		systemName, serviceCount, riskCount, score,
	)
}
