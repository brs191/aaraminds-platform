// Package obsplan implements the service-layer logic for the
// generate_observability_plan MCP tool.
//
// The service takes a structured description of services and produces a
// deterministic observability plan: per-service SLIs, SLOs derived from
// criticality and stated availability targets, recommended dashboards, and
// recommended alerts. It also reports coverage gaps (critical services with
// no alerts, API services with no latency SLI, missing availability target)
// and an overall observability-readiness score.
//
// The logic is deterministic and rule-based. No LLM calls, no external
// dependencies. Tests are table-driven and run in a few milliseconds.
package obsplan

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for generating an observability plan.
type Input struct {
	SystemName                string    `json:"system_name"`
	Description               string    `json:"description,omitempty"`
	Services                  []Service `json:"services"`
	NonFunctionalRequirements NFR       `json:"non_functional_requirements,omitempty"`
}

// Service describes a service to plan observability for.
type Service struct {
	Name          string `json:"name"`
	Criticality   string `json:"criticality,omitempty"` // high | medium | low (default medium)
	Type          string `json:"type,omitempty"`        // api | gateway | worker | datastore (default api)
	HasDashboards bool   `json:"has_dashboards,omitempty"`
	HasAlerts     bool   `json:"has_alerts,omitempty"`
}

// NFR captures non-functional requirements relevant to SLOs.
type NFR struct {
	AvailabilityTarget string `json:"availability_target,omitempty"` // e.g., "99.9"
	LatencyP99Ms       int    `json:"latency_p99_ms,omitempty"`
}

// Plan is the structured output: the observability plan.
type Plan struct {
	SystemName           string                 `json:"system_name"`
	ServiceObservability []ServiceObservability `json:"service_observability"`
	CoverageGaps         []Gap                  `json:"coverage_gaps"`
	ObservabilityScore   int                    `json:"observability_score"` // 0-100
	ObservabilityRating  string                 `json:"observability_rating"`
	Summary              string                 `json:"summary"`
}

// ServiceObservability is the plan for a single service.
type ServiceObservability struct {
	Service               string   `json:"service"`
	SLIs                  []SLI    `json:"slis"`
	SLOs                  []SLO    `json:"slos"`
	RecommendedDashboards []string `json:"recommended_dashboards"`
	RecommendedAlerts     []Alert  `json:"recommended_alerts"`
}

// SLI is a service-level indicator.
type SLI struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Measurement string `json:"measurement"`
}

// SLO is a service-level objective.
type SLO struct {
	SLI       string `json:"sli"`
	Objective string `json:"objective"`
	Window    string `json:"window"`
}

// Alert is a recommended alert.
type Alert struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Severity  string `json:"severity"` // page | ticket | info
}

// Gap is an observability coverage gap.
type Gap struct {
	Severity         string   `json:"severity"` // low | medium | high
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	ServicesAffected []string `json:"services_affected"`
	Recommendation   string   `json:"recommendation"`
}

// Planner is the observability-plan service.
type Planner struct{}

// NewService constructs a Planner.
func NewService() *Planner { return &Planner{} }

// Generate applies the rule set and returns a Plan.
func (p *Planner) Generate(in Input) (Plan, error) {
	if err := validate(in); err != nil {
		return Plan{}, err
	}

	plans := make([]ServiceObservability, 0, len(in.Services))
	for _, svc := range in.Services {
		plans = append(plans, planService(svc, in.NonFunctionalRequirements))
	}

	gaps := detectGaps(in)
	score := computeScore(gaps)
	rating := ratingForScore(score)

	return Plan{
		SystemName:           in.SystemName,
		ServiceObservability: plans,
		CoverageGaps:         gaps,
		ObservabilityScore:   score,
		ObservabilityRating:  rating,
		Summary:              composeSummary(in.SystemName, len(in.Services), len(gaps), score),
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

func criticalityOf(s Service) string {
	switch strings.ToLower(strings.TrimSpace(s.Criticality)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func typeOf(s Service) string {
	switch strings.ToLower(strings.TrimSpace(s.Type)) {
	case "gateway":
		return "gateway"
	case "worker":
		return "worker"
	case "datastore":
		return "datastore"
	default:
		return "api"
	}
}

func availabilityObjective(crit string) string {
	switch crit {
	case "high":
		return "99.95%"
	case "low":
		return "99.5%"
	default:
		return "99.9%"
	}
}

func alertSeverity(crit string) string {
	switch crit {
	case "high":
		return "page"
	case "low":
		return "info"
	default:
		return "ticket"
	}
}

func planService(svc Service, nfr NFR) ServiceObservability {
	crit := criticalityOf(svc)
	typ := typeOf(svc)

	slis := []SLI{
		{Name: "availability", Description: "Fraction of successful requests", Measurement: "1 - (5xx / total requests)"},
		{Name: "error_rate", Description: "Fraction of failed requests", Measurement: "(4xx+5xx) / total requests"},
	}
	if typ == "api" || typ == "gateway" {
		slis = append(slis, SLI{Name: "latency_p99", Description: "99th percentile request latency", Measurement: "histogram_quantile(0.99, request_duration_seconds)"})
	}
	if typ == "worker" {
		slis = append(slis,
			SLI{Name: "throughput", Description: "Processed messages per second", Measurement: "rate(messages_processed_total[5m])"},
			SLI{Name: "saturation", Description: "Queue backlog depth", Measurement: "queue_depth"},
		)
	}

	slos := []SLO{
		{SLI: "availability", Objective: availabilityObjective(crit), Window: "30d"},
	}
	if typ == "api" || typ == "gateway" {
		obj := "p99 < 500ms"
		if nfr.LatencyP99Ms > 0 {
			obj = fmt.Sprintf("p99 < %dms", nfr.LatencyP99Ms)
		}
		slos = append(slos, SLO{SLI: "latency_p99", Objective: obj, Window: "30d"})
	}

	dashboards := []string{
		fmt.Sprintf("%s — golden signals (latency, traffic, errors, saturation)", svc.Name),
		fmt.Sprintf("%s — SLO burn-down", svc.Name),
	}

	alerts := []Alert{
		{Name: fmt.Sprintf("%s availability SLO burn", svc.Name), Condition: "error budget burn rate > 2x over 1h", Severity: alertSeverity(crit)},
		{Name: fmt.Sprintf("%s elevated error rate", svc.Name), Condition: "error_rate > 5% for 10m", Severity: alertSeverity(crit)},
	}
	if typ == "api" || typ == "gateway" {
		alerts = append(alerts, Alert{Name: fmt.Sprintf("%s latency regression", svc.Name), Condition: "latency_p99 above SLO for 15m", Severity: alertSeverity(crit)})
	}
	if typ == "worker" {
		alerts = append(alerts, Alert{Name: fmt.Sprintf("%s queue backlog", svc.Name), Condition: "queue_depth growing for 15m", Severity: alertSeverity(crit)})
	}

	return ServiceObservability{
		Service:               svc.Name,
		SLIs:                  slis,
		SLOs:                  slos,
		RecommendedDashboards: dashboards,
		RecommendedAlerts:     alerts,
	}
}

func detectGaps(in Input) []Gap {
	gaps := []Gap{}

	if strings.TrimSpace(in.NonFunctionalRequirements.AvailabilityTarget) == "" {
		gaps = append(gaps, Gap{
			Severity:         "medium",
			Category:         "missing_availability_target",
			Description:      "no availability target is stated; SLOs default to criticality-based values that may not reflect business intent",
			ServicesAffected: []string{},
			Recommendation:   "Set an explicit availability target and error budget so SLOs and alert thresholds are anchored to a business commitment.",
		})
	}

	for _, svc := range in.Services {
		crit := criticalityOf(svc)
		typ := typeOf(svc)
		if !svc.HasAlerts {
			sev := "medium"
			if crit == "high" {
				sev = "high"
			}
			gaps = append(gaps, Gap{
				Severity:         sev,
				Category:         "no_alerts",
				Description:      fmt.Sprintf("service %q has no alerts configured; failures will be discovered by users, not operators", svc.Name),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Wire the recommended alerts (SLO burn, error rate, latency/backlog) to the on-call channel.",
			})
		}
		if !svc.HasDashboards {
			gaps = append(gaps, Gap{
				Severity:         "medium",
				Category:         "no_dashboards",
				Description:      fmt.Sprintf("service %q has no dashboards; incident triage will be slow without golden-signal visibility", svc.Name),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Stand up the recommended golden-signals and SLO burn-down dashboards before launch.",
			})
		}
		if (typ == "api" || typ == "gateway") && in.NonFunctionalRequirements.LatencyP99Ms == 0 {
			gaps = append(gaps, Gap{
				Severity:         "medium",
				Category:         "no_latency_target",
				Description:      fmt.Sprintf("%s service %q has no latency target; the latency SLO defaults to 500ms and may not match user expectations", typ, svc.Name),
				ServicesAffected: []string{svc.Name},
				Recommendation:   "Set latency_p99_ms from real user-experience requirements so the latency SLO is meaningful.",
			})
		}
	}

	sort.SliceStable(gaps, func(i, j int) bool {
		if gaps[i].Severity != gaps[j].Severity {
			return rank(gaps[i].Severity) > rank(gaps[j].Severity)
		}
		return gaps[i].Category < gaps[j].Category
	})
	return gaps
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

func computeScore(gaps []Gap) int {
	score := 100
	for _, g := range gaps {
		switch g.Severity {
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
		return "Observability is launch-ready; minor follow-ups only"
	case score >= 75:
		return "Observability is directionally sound; close targeted gaps before launch"
	case score >= 60:
		return "Observability has material gaps; address before production traffic"
	case score >= 40:
		return "Observability is insufficient; significant work needed before launch"
	default:
		return "Observability is largely absent; the system is not operable as designed"
	}
}

func composeSummary(systemName string, serviceCount, gapCount, score int) string {
	return fmt.Sprintf(
		"Observability plan for %s: %d services planned, %d coverage gaps, observability-readiness score %d/100.",
		systemName, serviceCount, gapCount, score,
	)
}
