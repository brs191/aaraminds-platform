// Package design implements the service-layer logic for three architecture
// review and recommendation MCP tools:
//
//   - recommend_microservice_patterns: takes a structured problem statement and
//     returns ranked pattern recommendations from a curated catalog, each with
//     rationale tied to signals in the input.
//   - review_microservice_design: walks the 9-dimension reviewer framework
//     (per the microservices-architecture-reviewer skill) over a structured
//     description of a system and produces per-dimension verdicts, named
//     defects, recommendations, an overall verdict, and a score.
//   - score_well_architected_readiness: scores the system against the five
//     Azure Well-Architected pillars (reliability, security, operational
//     excellence, performance efficiency, cost optimization) using deterministic
//     signal-based rules driven by the input.
//
// The logic is deterministic and rule-based. Given the same input, the output
// is byte-stable. No external dependencies, no LLM calls. Tests are
// table-driven and run in a few milliseconds.
//
// Conventions match the rest of the architecture-tools package set
// (boundary, archrisks, azuremap, adr, ...):
//
//   - Errors only for inputs that fundamentally cannot be processed
//     (empty system_name; no services where required).
//   - Weak shapes (no observability mention, no resilience controls, missing
//     SLOs, etc.) surface as findings, not errors.
//   - Severity / rating / scoring vocabularies are aligned across tools.
package design

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Service is the design review service. It hosts three independent
// rule-based capabilities (review, recommend, score). They share helpers but
// not state.
type Service struct{}

// NewService constructs a Service.
func NewService() *Service { return &Service{} }

// ---------------------------------------------------------------------------
// Shared input model
// ---------------------------------------------------------------------------

// ServiceDescriptor is one service in the proposed or existing system. The
// shape is intentionally permissive — every field is optional except Name —
// because the three tools each look at a different subset.
type ServiceDescriptor struct {
	Name        string   `json:"name"`
	Capability  string   `json:"capability,omitempty"`
	Stateful    bool     `json:"stateful,omitempty"`
	Replicated  bool     `json:"replicated,omitempty"`
	Criticality string   `json:"criticality,omitempty"` // high | medium | low
	DependsOn   []string `json:"depends_on,omitempty"`  // synchronous deps
	Async       []string `json:"async,omitempty"`       // async/event deps
	Resilience  []string `json:"resilience,omitempty"`  // timeout, retry, circuit_breaker, bulkhead, ...
	OwnsData    []string `json:"owns_data,omitempty"`
	Team        string   `json:"team,omitempty"`
}

// DataStoreDescriptor describes a data store referenced by the system.
type DataStoreDescriptor struct {
	Name           string `json:"name"`
	Kind           string `json:"kind,omitempty"`           // postgres, cosmos, redis, blob
	Encrypted      bool   `json:"encrypted,omitempty"`      // encryption at rest
	Classification string `json:"classification,omitempty"` // pii, phi, pci, sensitive, public
}

// NFR captures non-functional requirements relevant to scoring.
type NFR struct {
	AvailabilityTarget string `json:"availability_target,omitempty"` // e.g. "99.9"
	LatencyP99Ms       int    `json:"latency_p99_ms,omitempty"`
	RTOMinutes         int    `json:"rto_minutes,omitempty"`
	RPOMinutes         int    `json:"rpo_minutes,omitempty"`
}

// SystemInput is the shared, structured description used by the review and
// scoring tools. The pattern recommender uses its own narrower Input.
type SystemInput struct {
	SystemName         string                `json:"system_name"`
	BusinessCapability string                `json:"business_capability,omitempty"`
	DeploymentTarget   string                `json:"deployment_target,omitempty"` // aks | container_apps | app_service | functions | hybrid
	Services           []ServiceDescriptor   `json:"services,omitempty"`
	DataStores         []DataStoreDescriptor `json:"data_stores,omitempty"`

	// Cross-cutting capability flags. The reviewer and scorer interpret these
	// alongside per-service hints. They are optional; absence is itself a
	// finding rather than an error.
	Observability    []string `json:"observability,omitempty"`     // e.g. ["otel", "appinsights", "grafana"]
	SecurityControls []string `json:"security_controls,omitempty"` // e.g. ["entra_id", "managed_identity", "key_vault"]
	APIContracts     []string `json:"api_contracts,omitempty"`     // e.g. ["openapi", "versioning"]
	Messaging        []string `json:"messaging,omitempty"`         // e.g. ["service_bus", "event_hubs"]
	Patterns         []string `json:"patterns,omitempty"`          // patterns explicitly adopted

	Constraints []string `json:"constraints,omitempty"`
	NFR         NFR      `json:"non_functional_requirements,omitempty"`

	// AutoscaleDeclared and ScaleToZero are explicit opt-in cost/operability
	// signals. Together with NFR they drive the cost pillar.
	AutoscaleDeclared bool `json:"autoscale_declared,omitempty"`
	ScaleToZero       bool `json:"scale_to_zero,omitempty"`
	ReservedCapacity  bool `json:"reserved_capacity,omitempty"`

	// Backwards-compat: the previous Service.Review accepted a flat Services
	// []string. Callers that still pass that shape can populate this field
	// instead of Services; we lift it into Services on entry.
	ServicesFlat []string `json:"services_flat,omitempty"`
}

// ---------------------------------------------------------------------------
// review_microservice_design
// ---------------------------------------------------------------------------

// ReviewResult is the structured 9-dimension review output.
type ReviewResult struct {
	SystemName       string             `json:"system_name"`
	Verdict          string             `json:"verdict"` // Healthy | Healthy with risks | At risk | Unsound
	Score            int                `json:"score"`   // 0-100, derived from dimension ratings
	Summary          string             `json:"summary"`
	Dimensions       []DimensionFinding `json:"dimensions"`
	HardFails        []string           `json:"hard_fails"`
	SoftFails        []string           `json:"soft_fails"`
	MissingArtifacts []string           `json:"missing_artifacts"`
	NextSteps        []string           `json:"next_steps"`
}

// DimensionFinding is one of the nine review dimensions with a verdict,
// rationale, named defects, and a recommendation.
type DimensionFinding struct {
	Number         int      `json:"number"`         // 1..9
	Dimension      string   `json:"dimension"`      // human-readable label
	Rating         string   `json:"rating"`         // pass | soft_fail | hard_fail
	Note           string   `json:"note"`           // one-line rationale for the rating
	Defects        []string `json:"defects"`        // named, specific defects (empty on pass)
	Recommendation string   `json:"recommendation"` // smallest viable fix or affirming note
}

// Review walks the 9-dimension reviewer framework over the input and produces
// a structured verdict report. See the microservices-architecture-reviewer
// skill for the dimension definitions.
func (s *Service) Review(in SystemInput) (ReviewResult, error) {
	in = normalizeSystemInput(in)
	if err := validateSystemForReview(in); err != nil {
		return ReviewResult{}, err
	}

	dims := []DimensionFinding{
		reviewDimension1(in),
		reviewDimension2(in),
		reviewDimension3(in),
		reviewDimension4(in),
		reviewDimension5(in),
		reviewDimension6(in),
		reviewDimension7(in),
		reviewDimension8(in),
		reviewDimension9(in),
	}

	hardFails := []string{}
	softFails := []string{}
	for _, d := range dims {
		switch d.Rating {
		case "hard_fail":
			hardFails = append(hardFails, fmt.Sprintf("D%d %s: %s", d.Number, d.Dimension, d.Note))
		case "soft_fail":
			softFails = append(softFails, fmt.Sprintf("D%d %s: %s", d.Number, d.Dimension, d.Note))
		}
	}

	missing := missingArtifacts(in)
	score := reviewScore(dims, missing)
	verdict := reviewVerdict(len(hardFails), len(softFails), score)
	next := reviewNextSteps(dims)
	summary := fmt.Sprintf(
		"Architecture review of %s: %d services analyzed across 9 dimensions; %d hard-fail(s), %d soft-fail(s), score %d/100, verdict %q.",
		in.SystemName, len(in.Services), len(hardFails), len(softFails), score, verdict,
	)

	return ReviewResult{
		SystemName:       in.SystemName,
		Verdict:          verdict,
		Score:            score,
		Summary:          summary,
		Dimensions:       dims,
		HardFails:        hardFails,
		SoftFails:        softFails,
		MissingArtifacts: missing,
		NextSteps:        next,
	}, nil
}

func reviewDimension1(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 1, Dimension: "Domain & service boundaries"}
	defects := []string{}
	noCap := 0
	functionalNames := []string{}
	for _, svc := range in.Services {
		if strings.TrimSpace(svc.Capability) == "" {
			noCap++
		}
		ln := strings.ToLower(svc.Name)
		if ln == "userservice" || ln == "productservice" || ln == "orderservice" || ln == "datasservice" || strings.HasSuffix(ln, "service") && !strings.Contains(ln, "-") && len(strings.Fields(svc.Capability)) < 2 {
			functionalNames = append(functionalNames, svc.Name)
		}
	}
	if noCap == len(in.Services) && len(in.Services) > 0 {
		defects = append(defects, fmt.Sprintf("no business_capability declared for any of the %d services", len(in.Services)))
	} else if noCap > 0 {
		defects = append(defects, fmt.Sprintf("%d service(s) have no declared business_capability", noCap))
	}
	if len(functionalNames) > 0 {
		sort.Strings(functionalNames)
		defects = append(defects, fmt.Sprintf("services look functionally decomposed (technical-layer names) rather than capability-aligned: %s", strings.Join(functionalNames, ", ")))
	}
	if shared := sharedDataStores(in); len(shared) > 0 {
		defects = append(defects, fmt.Sprintf("shared data store(s) across services: %s", strings.Join(shared, ", ")))
	}
	switch {
	case len(defects) == 0:
		d.Rating = "pass"
		d.Note = "service set is capability-aligned and data ownership is exclusive"
		d.Recommendation = "no action required; revisit if a new bounded context is introduced"
	case anyShared(defects, "shared data store") || anyShared(defects, "functionally decomposed"):
		d.Rating = "hard_fail"
		d.Note = "boundary violations present (shared data or functional decomposition)"
		d.Recommendation = "redraw boundaries by business capability; give each data store exactly one owning service"
		d.Defects = defects
	default:
		d.Rating = "soft_fail"
		d.Note = "service set is partially defined; capability anchors missing on some services"
		d.Recommendation = "declare a business_capability per service so the bounded context is explicit"
		d.Defects = defects
	}
	return d
}

func reviewDimension2(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 2, Dimension: "Data architecture & consistency"}
	defects := []string{}
	mentionsSaga := mentionsAny(in.Patterns, "saga")
	mentionsOutbox := mentionsAny(in.Patterns, "outbox") || mentionsAny(in.Patterns, "transactional_outbox")
	hasIdempotency := mentionsAny(in.Patterns, "idempotent") || mentionsAny(in.Patterns, "idempotency")
	crossServiceWrites := false
	for _, svc := range in.Services {
		if len(svc.DependsOn) > 0 {
			crossServiceWrites = true
			break
		}
	}
	if crossServiceWrites && !mentionsSaga {
		defects = append(defects, "cross-service writes implied by synchronous dependencies but no saga/outbox pattern declared")
	}
	if mentionsAny(in.Patterns, "2pc") || mentionsAny(in.Patterns, "two_phase_commit") {
		defects = append(defects, "distributed two-phase commit declared; this is a hard-fail for cloud-native microservices")
	}
	if crossServiceWrites && mentionsSaga && !mentionsOutbox {
		defects = append(defects, "saga declared but no transactional outbox — events can be lost on broker failure")
	}
	if crossServiceWrites && !hasIdempotency {
		defects = append(defects, "no idempotency guard on state-changing cross-service calls")
	}
	switch {
	case anyShared(defects, "two-phase commit"):
		d.Rating = "hard_fail"
		d.Note = "distributed 2PC is named; this is unacceptable in a cloud-native estate"
		d.Recommendation = "replace 2PC with saga + transactional outbox + idempotent consumers"
		d.Defects = defects
	case len(defects) >= 2:
		d.Rating = "hard_fail"
		d.Note = "cross-service consistency is not engineered (no saga + no idempotency + no outbox)"
		d.Recommendation = "introduce saga with transactional outbox; require idempotency keys on every write path"
		d.Defects = defects
	case len(defects) == 1:
		d.Rating = "soft_fail"
		d.Note = "data architecture is mostly addressed; one consistency gap remains"
		d.Recommendation = "close the named gap; document the consistency model per cross-service write"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "data ownership is exclusive and cross-service consistency is modeled"
		d.Recommendation = "no action; reassess if a new write path crosses a service boundary"
	}
	return d
}

func reviewDimension3(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 3, Dimension: "Communication topology"}
	defects := []string{}
	deepest := 0
	deepName := ""
	for _, svc := range in.Services {
		if n := len(svc.DependsOn); n > deepest {
			deepest = n
			deepName = svc.Name
		}
	}
	if deepest >= 4 {
		defects = append(defects, fmt.Sprintf("synchronous chain of %d hops at service %q — slow dependency cascades", deepest, deepName))
	}
	if len(in.Messaging) == 0 && hasCrossServiceAsync(in) {
		defects = append(defects, "async dependencies declared but no broker named in messaging[]")
	}
	if len(in.Messaging) > 0 && !mentionsAny(in.Patterns, "dlq") && !mentionsAny(in.Patterns, "dead_letter") {
		defects = append(defects, "broker present but no DLQ pattern declared — poison messages will block consumers")
	}
	switch {
	case deepest >= 4:
		d.Rating = "hard_fail"
		d.Note = "synchronous chain depth ≥4 is a load-bearing failure mode"
		d.Recommendation = fmt.Sprintf("convert non-essential edges of %q's call chain to async events via Service Bus", deepName)
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "messaging is in use but operational hygiene is incomplete"
		d.Recommendation = "name the broker (Service Bus or Event Hubs) and configure DLQs on every critical async path"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "sync/async split is intentional; topology is shallow"
		d.Recommendation = "no action; reassess if a new sync dependency is added"
	}
	return d
}

func reviewDimension4(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 4, Dimension: "API contracts"}
	defects := []string{}
	hasContracts := false
	hasVersioning := false
	for _, c := range in.APIContracts {
		l := strings.ToLower(c)
		if strings.Contains(l, "openapi") || strings.Contains(l, "proto") || strings.Contains(l, "grpc") {
			hasContracts = true
		}
		if strings.Contains(l, "versioning") || strings.Contains(l, "version") {
			hasVersioning = true
		}
	}
	if !hasContracts {
		defects = append(defects, "no API contract format declared (expected openapi or proto/grpc)")
	}
	if !hasVersioning {
		defects = append(defects, "no API versioning strategy declared")
	}
	switch {
	case !hasContracts && !hasVersioning:
		d.Rating = "hard_fail"
		d.Note = "neither contract format nor versioning strategy is declared"
		d.Recommendation = "publish OpenAPI per service and adopt URI or header-based versioning before the next consumer integrates"
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "API contract surface is partially defined"
		d.Recommendation = "close the named contract gap; gate breaking changes on a version bump"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "contracts and versioning are declared"
		d.Recommendation = "no action; keep contract reviews in PRs"
	}
	return d
}

func reviewDimension5(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 5, Dimension: "Resilience"}
	defects := []string{}
	critWithoutResilience := []string{}
	syncWithoutResilience := []string{}
	for _, svc := range in.Services {
		hasRes := len(svc.Resilience) > 0
		if criticalityOf(svc) == "high" && !hasRes {
			critWithoutResilience = append(critWithoutResilience, svc.Name)
		}
		if len(svc.DependsOn) > 0 && !hasRes {
			syncWithoutResilience = append(syncWithoutResilience, svc.Name)
		}
	}
	if len(critWithoutResilience) > 0 {
		sort.Strings(critWithoutResilience)
		defects = append(defects, fmt.Sprintf("high-criticality service(s) declare no resilience controls: %s", strings.Join(critWithoutResilience, ", ")))
	}
	if len(syncWithoutResilience) > len(critWithoutResilience) {
		extra := diff(syncWithoutResilience, critWithoutResilience)
		sort.Strings(extra)
		if len(extra) > 0 {
			defects = append(defects, fmt.Sprintf("synchronous-caller service(s) without resilience controls: %s", strings.Join(extra, ", ")))
		}
	}
	switch {
	case len(critWithoutResilience) > 0:
		d.Rating = "hard_fail"
		d.Note = "critical services lack timeout/retry/circuit-breaker policies"
		d.Recommendation = "add explicit timeout, retry-with-jitter, and circuit-breaker policies on every outbound call from critical services"
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "non-critical synchronous callers lack resilience controls"
		d.Recommendation = "apply the standard resilience policy bundle (timeout + retry + breaker) consistently across all sync callers"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "resilience controls present on services with synchronous outbound calls"
		d.Recommendation = "no action; validate breaker thresholds in load tests quarterly"
	}
	return d
}

func reviewDimension6(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 6, Dimension: "Azure service mapping"}
	defects := []string{}
	target := strings.ToLower(strings.TrimSpace(in.DeploymentTarget))
	allowed := map[string]bool{"aks": true, "container_apps": true, "app_service": true, "functions": true, "hybrid": true}
	if target == "" {
		defects = append(defects, "no deployment target declared (expected aks | container_apps | app_service | functions | hybrid)")
	} else if !allowed[target] {
		defects = append(defects, fmt.Sprintf("deployment target %q is not an Azure-native option from the catalog", in.DeploymentTarget))
	}
	for _, c := range append(append([]string{}, in.Patterns...), in.Messaging...) {
		lc := strings.ToLower(c)
		if strings.Contains(lc, "kafka") && !strings.Contains(lc, "event_hubs") {
			defects = append(defects, fmt.Sprintf("non-Azure component %q named; prefer Event Hubs (Kafka API) for an Azure-native stack", c))
		}
		if strings.Contains(lc, "sqs") || strings.Contains(lc, "kinesis") || strings.Contains(lc, "lambda") {
			defects = append(defects, fmt.Sprintf("AWS service drift: %q is not in the Azure stack", c))
		}
	}
	switch {
	case anyShared(defects, "AWS service drift") || anyShared(defects, "not an Azure-native option"):
		d.Rating = "hard_fail"
		d.Note = "non-Azure components named in the design"
		d.Recommendation = "replace with the equivalent Azure service from the catalog (Service Bus, Event Hubs, Functions, etc.)"
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "Azure mapping is mostly aligned but incomplete"
		d.Recommendation = "declare the deployment target and confirm every component maps to a verified Azure service"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "deployment target is Azure-native and no service drift detected"
		d.Recommendation = "no action; reassess if a new component is introduced"
	}
	return d
}

func reviewDimension7(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 7, Dimension: "Observability"}
	defects := []string{}
	hasOtel := mentionsAny(in.Observability, "otel") || mentionsAny(in.Observability, "opentelemetry")
	hasAPM := mentionsAny(in.Observability, "appinsights") || mentionsAny(in.Observability, "application_insights") || mentionsAny(in.Observability, "azure_monitor")
	hasSLO := strings.TrimSpace(in.NFR.AvailabilityTarget) != ""
	if !hasOtel {
		defects = append(defects, "no OpenTelemetry instrumentation declared")
	}
	if !hasAPM {
		defects = append(defects, "no APM target declared (Application Insights or Azure Monitor)")
	}
	if !hasSLO {
		defects = append(defects, "no availability SLO declared (non_functional_requirements.availability_target)")
	}
	switch {
	case !hasOtel && !hasSLO:
		d.Rating = "hard_fail"
		d.Note = "system is not operable to an SLO — no tracing and no target"
		d.Recommendation = "instrument every service with OpenTelemetry, export to Application Insights, and set a per-service SLO"
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "observability surface is partial"
		d.Recommendation = "close the named observability gap before production traffic"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "tracing, APM, and SLO are declared"
		d.Recommendation = "no action; review SLO error-budget burn monthly"
	}
	return d
}

func reviewDimension8(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 8, Dimension: "Security & compliance"}
	defects := []string{}
	hasIdentity := mentionsAny(in.SecurityControls, "entra") || mentionsAny(in.SecurityControls, "aad") || mentionsAny(in.SecurityControls, "oauth")
	hasMI := mentionsAny(in.SecurityControls, "managed_identity") || mentionsAny(in.SecurityControls, "managed-identity")
	hasKV := mentionsAny(in.SecurityControls, "key_vault") || mentionsAny(in.SecurityControls, "keyvault")
	if !hasIdentity {
		defects = append(defects, "no identity provider declared (expected Entra ID)")
	}
	if !hasMI {
		defects = append(defects, "no managed identity declared for service-to-service auth")
	}
	if !hasKV {
		defects = append(defects, "no secret store declared (expected Azure Key Vault)")
	}
	for _, ds := range in.DataStores {
		c := strings.ToLower(strings.TrimSpace(ds.Classification))
		if (c == "pii" || c == "phi" || c == "pci" || c == "sensitive") && !ds.Encrypted {
			defects = append(defects, fmt.Sprintf("data store %q is classified %q but is not encrypted at rest", ds.Name, ds.Classification))
		}
	}
	switch {
	case anyShared(defects, "not encrypted at rest") || (!hasIdentity && !hasKV):
		d.Rating = "hard_fail"
		d.Note = "missing identity/secret store or sensitive data unencrypted at rest"
		d.Recommendation = "enable Entra ID + managed identity, route all secrets through Key Vault, enable encryption at rest on every sensitive store"
		d.Defects = defects
	case len(defects) > 0:
		d.Rating = "soft_fail"
		d.Note = "security baseline is partially in place"
		d.Recommendation = "close the named security gap before exposing the system to production traffic"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "identity, managed identity, and Key Vault are declared"
		d.Recommendation = "no action; map SOC 2 / ISO 27001 controls as part of the compliance evidence cycle"
	}
	return d
}

func reviewDimension9(in SystemInput) DimensionFinding {
	d := DimensionFinding{Number: 9, Dimension: "Cost & operability"}
	defects := []string{}
	if !in.AutoscaleDeclared {
		defects = append(defects, "no autoscaling declared — sizing is static and either over- or under-provisioned")
	}
	if !in.ScaleToZero && !in.ReservedCapacity {
		defects = append(defects, "neither scale-to-zero nor reserved capacity declared — cost shape is undefined")
	}
	if in.NFR.RTOMinutes == 0 && in.NFR.RPOMinutes == 0 {
		defects = append(defects, "no RTO/RPO declared — recoverability targets undefined")
	}
	switch {
	case len(defects) >= 2:
		d.Rating = "soft_fail"
		d.Note = "cost/operability posture is incomplete"
		d.Recommendation = "declare autoscaling and choose scale-to-zero (spiky) or reserved capacity (steady) per workload; set RTO/RPO per critical service"
		d.Defects = defects
	case len(defects) == 1:
		d.Rating = "soft_fail"
		d.Note = "one cost/operability dimension undefined"
		d.Recommendation = "close the named cost or recoverability gap"
		d.Defects = defects
	default:
		d.Rating = "pass"
		d.Note = "autoscaling and capacity strategy are declared; recovery targets are set"
		d.Recommendation = "no action; review cost monthly against the budget"
	}
	return d
}

func missingArtifacts(in SystemInput) []string {
	miss := []string{}
	if len(in.APIContracts) == 0 {
		miss = append(miss, "API contracts (OpenAPI / proto) not provided")
	}
	if len(in.Observability) == 0 {
		miss = append(miss, "observability stack not declared")
	}
	if strings.TrimSpace(in.NFR.AvailabilityTarget) == "" {
		miss = append(miss, "availability SLO not declared")
	}
	if in.NFR.RTOMinutes == 0 && in.NFR.RPOMinutes == 0 {
		miss = append(miss, "recovery objectives (RTO/RPO) not stated")
	}
	return miss
}

func reviewScore(dims []DimensionFinding, missing []string) int {
	score := 100
	for _, d := range dims {
		switch d.Rating {
		case "hard_fail":
			score -= 12
		case "soft_fail":
			score -= 5
		}
	}
	score -= 2 * len(missing)
	if score < 0 {
		score = 0
	}
	return score
}

func reviewVerdict(hardFails, softFails, score int) string {
	switch {
	case hardFails == 0 && softFails == 0:
		return "Healthy"
	case hardFails == 0:
		return "Healthy with named risks"
	case hardFails <= 2 && score >= 50:
		return "At risk (named blockers)"
	default:
		return "Architecturally unsound (rebuild path required)"
	}
}

func reviewNextSteps(dims []DimensionFinding) []string {
	steps := []string{}
	seen := map[string]bool{}
	// Hard fails first, then soft fails, deduped.
	for _, rating := range []string{"hard_fail", "soft_fail"} {
		for _, d := range dims {
			if d.Rating != rating {
				continue
			}
			rec := d.Recommendation
			if seen[rec] || rec == "" {
				continue
			}
			seen[rec] = true
			steps = append(steps, fmt.Sprintf("[D%d %s] %s", d.Number, d.Dimension, rec))
			if len(steps) == 6 {
				return steps
			}
		}
	}
	return steps
}

// ---------------------------------------------------------------------------
// recommend_microservice_patterns
// ---------------------------------------------------------------------------

// PatternRecommendInput is the structured request for pattern recommendations.
type PatternRecommendInput struct {
	SystemName       string   `json:"system_name,omitempty"`
	Problem          string   `json:"problem"`
	DeploymentTarget string   `json:"deployment_target,omitempty"`
	Constraints      []string `json:"constraints,omitempty"`
}

// PatternRecommendation is one suggested pattern with rationale.
type PatternRecommendation struct {
	Pattern   string   `json:"pattern"`
	Category  string   `json:"category"`
	Rationale string   `json:"rationale"`
	Triggers  []string `json:"triggers"` // input keywords that activated this rule
	Skill     string   `json:"skill"`    // pack skill that owns this pattern
}

// PatternRecommendResult is the structured output.
type PatternRecommendResult struct {
	SystemName      string                  `json:"system_name,omitempty"`
	Recommendations []PatternRecommendation `json:"recommendations"`
	NotRecommended  []string                `json:"not_recommended,omitempty"`
	Summary         string                  `json:"summary"`
}

// RecommendPatterns parses the problem statement and proposes patterns from
// the curated catalog. Each recommendation cites the input phrases that
// triggered it so the user can audit the reasoning.
func (s *Service) RecommendPatterns(in PatternRecommendInput) (PatternRecommendResult, error) {
	if strings.TrimSpace(in.Problem) == "" {
		return PatternRecommendResult{}, errors.New("problem is required")
	}
	lc := strings.ToLower(in.Problem)

	type rule struct {
		pattern   string
		category  string
		triggers  []string
		rationale string
		skill     string
	}

	rules := []rule{
		{"api_gateway", "ingress", []string{"public api", "api gateway", "edge", "rate limit", "throttle"},
			"Expose a managed gateway in front of the services so authentication, throttling, and routing are enforced consistently. Map to Azure API Management.",
			"microservices-api-design"},
		{"saga", "consistency", []string{"transaction", "workflow", "across services", "compensation", "long-running"},
			"Use a saga pattern with explicit compensation steps so cross-service workflows recover deterministically without distributed 2PC. Implement as Durable Functions or a coded orchestrator.",
			"microservices-data-architecture"},
		{"transactional_outbox", "consistency", []string{"transaction", "events", "publish", "consistency", "outbox"},
			"Pair the local DB write with an outbox table so events are published atomically with state changes — eliminates the dual-write failure mode.",
			"microservices-data-architecture"},
		{"idempotent_consumer", "consistency", []string{"retry", "duplicate", "exactly once", "idempoten"},
			"Make consumers idempotent (idempotency key or natural-key dedupe) so safe retries do not double-apply state.",
			"microservices-data-architecture"},
		{"cqrs", "data", []string{"read", "query", "report", "scale read", "different read"},
			"Separate the write model from the read model so each scales and is shaped independently. Use Azure SQL for writes and Cosmos DB for reads when access patterns diverge.",
			"microservices-data-architecture"},
		{"event_sourcing", "data", []string{"audit", "history", "rebuild state", "event sourcing", "replay"},
			"Store the system of record as an append-only event log so any projection can be rebuilt and the full history is auditable.",
			"microservices-data-architecture"},
		{"cache_aside", "performance", []string{"cache", "hot read", "slow read", "latency"},
			"Front read-heavy paths with a cache-aside layer (Azure Cache for Redis) to absorb hot reads and reduce database pressure.",
			"microservices-data-architecture"},
		{"circuit_breaker", "resilience", []string{"timeout", "slow dependency", "cascading", "vendor", "outage"},
			"Wrap synchronous outbound calls in a circuit breaker so a slow or failing dependency does not exhaust the caller's thread pool.",
			"microservices-resilience"},
		{"bulkhead", "resilience", []string{"isolation", "noisy neighbor", "bulkhead", "thread pool"},
			"Isolate critical workloads in dedicated thread or connection pools so one slow path cannot starve the rest.",
			"microservices-resilience"},
		{"retry_with_jitter", "resilience", []string{"retry", "transient", "flaky", "rate limit"},
			"Retry transient failures with exponential backoff and jitter to avoid synchronized retry storms.",
			"microservices-resilience"},
		{"rate_limiting", "resilience", []string{"abuse", "throttle", "rate limit", "ddos", "burst"},
			"Throttle inbound traffic at the gateway (APIM) and at each service to protect downstream capacity.",
			"microservices-resilience"},
		{"async_messaging", "topology", []string{"async", "decouple", "queue", "fan out", "background"},
			"Move non-blocking work behind a durable broker (Azure Service Bus) so producers and consumers scale independently and failures are isolated.",
			"microservices-async-messaging"},
		{"event_streaming", "topology", []string{"stream", "telemetry", "high throughput", "kafka", "event hubs"},
			"Use a high-throughput streaming platform (Azure Event Hubs, Kafka-compatible) for partitioned, replayable event distribution.",
			"microservices-async-messaging"},
		{"strangler_fig", "migration", []string{"monolith", "legacy", "extract", "migrate", "modernize"},
			"Extract bounded contexts behind the gateway one at a time so the legacy and new systems coexist during migration; cut over per route, not per release.",
			"microservices-architecture-design"},
		{"blue_green", "rollout", []string{"deploy", "rollout", "zero downtime", "blue green"},
			"Use blue-green deployment so the new version runs in parallel and traffic is flipped only after validation; rollback is instant.",
			"microservices-resilience"},
		{"canary", "rollout", []string{"canary", "gradual rollout", "progressive delivery", "feature flag"},
			"Roll out gradually to a small traffic slice first and watch SLO/error-budget signals before widening.",
			"microservices-resilience"},
		{"sidecar", "platform", []string{"cross-cutting", "polyglot", "service mesh", "sidecar"},
			"Push cross-cutting concerns (mTLS, retries, metrics) into a sidecar (service mesh) so multi-language services stay consistent.",
			"microservices-async-messaging"},
		{"backend_for_frontend", "ingress", []string{"mobile", "web", "frontend", "bff"},
			"Add a BFF per client channel so each frontend gets a contract shaped to its needs without bloating the core services.",
			"microservices-api-design"},
		{"saga_orchestration", "consistency", []string{"orchestrator", "central workflow", "durable", "process manager"},
			"Run the saga via a central orchestrator (Durable Functions) when steps need explicit state and visibility; prefer choreography only when steps are independent.",
			"microservices-data-architecture"},
	}

	recs := []PatternRecommendation{}
	seen := map[string]bool{}
	for _, r := range rules {
		matched := matchedTriggers(lc, r.triggers)
		if len(matched) == 0 {
			continue
		}
		if seen[r.pattern] {
			continue
		}
		seen[r.pattern] = true
		recs = append(recs, PatternRecommendation{
			Pattern:   r.pattern,
			Category:  r.category,
			Triggers:  matched,
			Rationale: r.rationale,
			Skill:     r.skill,
		})
	}

	// Cross-cutting defaults: API gateway and observability are almost always
	// the right baseline when the user is asking about microservices at all.
	if !seen["api_gateway"] {
		recs = append(recs, PatternRecommendation{
			Pattern:   "api_gateway",
			Category:  "ingress",
			Triggers:  []string{"<default>"},
			Rationale: "Default: every public-facing microservices estate needs a gateway (APIM) for auth, throttling, and routing.",
			Skill:     "microservices-api-design",
		})
	}

	// Sort: catalog order is by category, then pattern name.
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].Category != recs[j].Category {
			return recs[i].Category < recs[j].Category
		}
		return recs[i].Pattern < recs[j].Pattern
	})

	notRec := []string{}
	if strings.Contains(lc, "2pc") || strings.Contains(lc, "two-phase commit") || strings.Contains(lc, "distributed transaction") {
		notRec = append(notRec, "two_phase_commit — incompatible with cloud-native microservices; use saga + outbox instead")
	}
	if strings.Contains(lc, "shared database") || strings.Contains(lc, "single database") {
		notRec = append(notRec, "shared_database — violates service ownership; give each service its own store")
	}

	summary := fmt.Sprintf(
		"Recommended %d pattern(s) for problem of %d chars; %d anti-patterns flagged.",
		len(recs), len(in.Problem), len(notRec),
	)
	if strings.TrimSpace(in.SystemName) != "" {
		summary = fmt.Sprintf("[%s] %s", in.SystemName, summary)
	}

	return PatternRecommendResult{
		SystemName:      in.SystemName,
		Recommendations: recs,
		NotRecommended:  notRec,
		Summary:         summary,
	}, nil
}

func matchedTriggers(haystack string, needles []string) []string {
	out := []string{}
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			out = append(out, n)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// score_well_architected_readiness
// ---------------------------------------------------------------------------

// PillarScore is one Well-Architected pillar with its score, evidence, and risks.
type PillarScore struct {
	Score    int      `json:"score"`
	Evidence []string `json:"evidence"`
	Risks    []string `json:"risks"`
}

// Scorecard is the structured Well-Architected output.
type Scorecard struct {
	SystemName            string      `json:"system_name"`
	Reliability           PillarScore `json:"reliability"`
	Security              PillarScore `json:"security"`
	OperationalExcellence PillarScore `json:"operational_excellence"`
	PerformanceEfficiency PillarScore `json:"performance_efficiency"`
	CostOptimization      PillarScore `json:"cost_optimization"`
	OverallScore          int         `json:"overall_score"`
	Rating                string      `json:"rating"`
	Summary               string      `json:"summary"`
}

// ScoreWellArchitected scores the system against the five Azure Well-Architected
// pillars. Each pillar's score is derived from explicit input signals
// (services count, observability mentions, security controls, NFR targets,
// resilience controls, autoscaling, etc.) — there are no hardcoded scores.
func (s *Service) ScoreWellArchitected(in SystemInput) (Scorecard, error) {
	in = normalizeSystemInput(in)
	if strings.TrimSpace(in.SystemName) == "" {
		return Scorecard{}, errors.New("system_name is required")
	}

	rel := scoreReliability(in)
	sec := scoreSecurity(in)
	ops := scoreOperationalExcellence(in)
	perf := scorePerformanceEfficiency(in)
	cost := scoreCostOptimization(in)

	overall := (rel.Score + sec.Score + ops.Score + perf.Score + cost.Score) / 5
	rating := wellArchitectedRating(overall)

	return Scorecard{
		SystemName:            in.SystemName,
		Reliability:           rel,
		Security:              sec,
		OperationalExcellence: ops,
		PerformanceEfficiency: perf,
		CostOptimization:      cost,
		OverallScore:          overall,
		Rating:                rating,
		Summary: fmt.Sprintf(
			"Well-Architected scorecard for %s: reliability %d, security %d, ops %d, perf %d, cost %d; overall %d/100 (%s).",
			in.SystemName, rel.Score, sec.Score, ops.Score, perf.Score, cost.Score, overall, rating,
		),
	}, nil
}

func scoreReliability(in SystemInput) PillarScore {
	score := 60
	ev := []string{}
	rk := []string{}

	if mentionsAny(in.Patterns, "saga") {
		score += 8
		ev = append(ev, "saga declared")
	}
	if mentionsAny(in.Patterns, "outbox") || mentionsAny(in.Patterns, "transactional_outbox") {
		score += 6
		ev = append(ev, "transactional outbox declared")
	}
	if mentionsAny(in.Patterns, "dlq") || mentionsAny(in.Patterns, "dead_letter") {
		score += 4
		ev = append(ev, "DLQ declared")
	}
	resilienceCovered := 0
	for _, svc := range in.Services {
		if len(svc.Resilience) > 0 {
			resilienceCovered++
		}
	}
	if len(in.Services) > 0 {
		ratio := float64(resilienceCovered) / float64(len(in.Services))
		score += int(ratio * 15)
		ev = append(ev, fmt.Sprintf("%d/%d services declare resilience controls", resilienceCovered, len(in.Services)))
		if ratio < 0.5 {
			rk = append(rk, "more than half of services lack timeout/retry/circuit-breaker policies")
		}
	}
	if strings.TrimSpace(in.NFR.AvailabilityTarget) != "" {
		score += 5
		ev = append(ev, "availability SLO declared: "+in.NFR.AvailabilityTarget)
	} else {
		rk = append(rk, "no availability SLO declared; resilience investment cannot be sized")
	}
	if in.NFR.RTOMinutes > 0 || in.NFR.RPOMinutes > 0 {
		score += 4
		ev = append(ev, fmt.Sprintf("recovery objectives set (RTO=%dm, RPO=%dm)", in.NFR.RTOMinutes, in.NFR.RPOMinutes))
	} else {
		rk = append(rk, "no RTO/RPO declared")
	}
	if len(in.Services) >= 5 && !hasReplicatedCriticalServices(in) {
		score -= 6
		rk = append(rk, "no critical service is marked replicated; single-instance services are brittle in a 5+ service estate")
	}

	return clampPillar(score, ev, rk)
}

func scoreSecurity(in SystemInput) PillarScore {
	score := 50
	ev := []string{}
	rk := []string{}

	controls := map[string]bool{}
	for _, c := range in.SecurityControls {
		controls[strings.ToLower(strings.TrimSpace(c))] = true
	}
	if controls["entra_id"] || controls["entra"] || controls["aad"] || controls["oauth"] {
		score += 10
		ev = append(ev, "identity provider declared (Entra ID / OAuth)")
	} else {
		rk = append(rk, "no identity provider declared")
	}
	if controls["managed_identity"] || controls["managed-identity"] {
		score += 10
		ev = append(ev, "managed identity declared for service-to-service auth")
	} else {
		rk = append(rk, "no managed identity for service-to-service auth")
	}
	if controls["key_vault"] || controls["keyvault"] {
		score += 10
		ev = append(ev, "Key Vault declared as the secret store")
	} else {
		rk = append(rk, "no Key Vault declared; secrets likely handled ad hoc")
	}
	if controls["mtls"] || controls["service_mesh"] {
		score += 5
		ev = append(ev, "mTLS / service mesh declared")
	}
	if controls["waf"] || controls["azure_waf"] || controls["front_door_waf"] {
		score += 5
		ev = append(ev, "WAF declared at the edge")
	}

	sensitive := 0
	encryptedSensitive := 0
	for _, ds := range in.DataStores {
		c := strings.ToLower(strings.TrimSpace(ds.Classification))
		if c == "pii" || c == "phi" || c == "pci" || c == "sensitive" {
			sensitive++
			if ds.Encrypted {
				encryptedSensitive++
			}
		}
	}
	if sensitive > 0 {
		if encryptedSensitive == sensitive {
			score += 10
			ev = append(ev, fmt.Sprintf("all %d sensitive data store(s) encrypted at rest", sensitive))
		} else {
			score -= 10 * (sensitive - encryptedSensitive)
			rk = append(rk, fmt.Sprintf("%d sensitive data store(s) unencrypted at rest", sensitive-encryptedSensitive))
		}
	}

	if anyConstraintMentions(in.Constraints, "soc2", "soc 2", "iso27001", "iso 27001", "hipaa", "pci", "gdpr") {
		ev = append(ev, "compliance constraint declared in constraints[]")
	}

	return clampPillar(score, ev, rk)
}

func scoreOperationalExcellence(in SystemInput) PillarScore {
	score := 55
	ev := []string{}
	rk := []string{}

	if mentionsAny(in.Observability, "otel") || mentionsAny(in.Observability, "opentelemetry") {
		score += 10
		ev = append(ev, "OpenTelemetry instrumentation declared")
	} else {
		rk = append(rk, "no OpenTelemetry instrumentation declared")
	}
	if mentionsAny(in.Observability, "appinsights") || mentionsAny(in.Observability, "application_insights") || mentionsAny(in.Observability, "azure_monitor") {
		score += 8
		ev = append(ev, "Azure Monitor / Application Insights declared as APM")
	} else {
		rk = append(rk, "no APM target declared (Application Insights / Azure Monitor)")
	}
	if mentionsAny(in.Observability, "grafana") || mentionsAny(in.Observability, "prometheus") {
		score += 4
		ev = append(ev, "Grafana / Prometheus declared")
	}
	if mentionsAny(in.Patterns, "ci_cd") || mentionsAny(in.Patterns, "github_actions") || mentionsAny(in.Patterns, "pipeline") {
		score += 4
		ev = append(ev, "CI/CD pipeline declared")
	}
	ownedServices := 0
	for _, svc := range in.Services {
		if strings.TrimSpace(svc.Team) != "" {
			ownedServices++
		}
	}
	if len(in.Services) > 0 {
		ratio := float64(ownedServices) / float64(len(in.Services))
		score += int(ratio * 10)
		ev = append(ev, fmt.Sprintf("%d/%d services have a named owning team", ownedServices, len(in.Services)))
		if ratio < 1.0 {
			rk = append(rk, fmt.Sprintf("%d service(s) have no owning team; ownership drift expected", len(in.Services)-ownedServices))
		}
	}
	if strings.TrimSpace(in.DeploymentTarget) == "" {
		score -= 5
		rk = append(rk, "deployment target not declared")
	}

	return clampPillar(score, ev, rk)
}

func scorePerformanceEfficiency(in SystemInput) PillarScore {
	score := 60
	ev := []string{}
	rk := []string{}

	if in.AutoscaleDeclared {
		score += 10
		ev = append(ev, "autoscaling declared")
	} else {
		rk = append(rk, "autoscaling not declared")
	}
	if mentionsAny(in.Patterns, "cache") || mentionsAny(in.Patterns, "cache_aside") || mentionsAny(in.Patterns, "redis") {
		score += 8
		ev = append(ev, "cache pattern declared")
	}
	if len(in.Messaging) > 0 {
		score += 6
		ev = append(ev, "async messaging declared ("+strings.Join(in.Messaging, ", ")+")")
	}
	if in.NFR.LatencyP99Ms > 0 {
		score += 5
		ev = append(ev, fmt.Sprintf("latency target declared (p99=%dms)", in.NFR.LatencyP99Ms))
	} else {
		rk = append(rk, "no latency target declared; performance work cannot be sized")
	}
	if mentionsAny(in.Patterns, "cqrs") || mentionsAny(in.Patterns, "read_model") {
		score += 5
		ev = append(ev, "CQRS / read model declared")
	}
	// Penalise deep sync chains as a perf risk too.
	deepest := 0
	for _, svc := range in.Services {
		if n := len(svc.DependsOn); n > deepest {
			deepest = n
		}
	}
	if deepest >= 4 {
		score -= 8
		rk = append(rk, fmt.Sprintf("synchronous chain depth %d adds tail-latency risk", deepest))
	}

	return clampPillar(score, ev, rk)
}

func scoreCostOptimization(in SystemInput) PillarScore {
	score := 55
	ev := []string{}
	rk := []string{}

	if in.AutoscaleDeclared {
		score += 10
		ev = append(ev, "autoscaling declared (sizing follows demand)")
	} else {
		rk = append(rk, "no autoscaling — static sizing tends to over- or under-provision")
	}
	if in.ScaleToZero {
		score += 8
		ev = append(ev, "scale-to-zero declared for spiky workloads")
	}
	if in.ReservedCapacity {
		score += 8
		ev = append(ev, "reserved capacity declared for steady-state baseline")
	}
	if !in.ScaleToZero && !in.ReservedCapacity {
		rk = append(rk, "neither scale-to-zero nor reserved capacity — cost shape undefined")
	}
	target := strings.ToLower(strings.TrimSpace(in.DeploymentTarget))
	if target == "container_apps" || target == "functions" {
		score += 5
		ev = append(ev, "deployment target supports scale-to-zero economics")
	}
	// More services = more cost surface; flag when no observability for cost.
	if len(in.Services) >= 8 && !mentionsAny(in.Observability, "cost") && !mentionsAny(in.Observability, "azure_monitor") {
		score -= 5
		rk = append(rk, fmt.Sprintf("%d services with no cost-visibility signal", len(in.Services)))
	}

	return clampPillar(score, ev, rk)
}

func clampPillar(score int, ev []string, rk []string) PillarScore {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	if ev == nil {
		ev = []string{}
	}
	if rk == nil {
		rk = []string{}
	}
	return PillarScore{Score: score, Evidence: ev, Risks: rk}
}

func wellArchitectedRating(score int) string {
	switch {
	case score >= 90:
		return "Strong — production-ready across all five pillars"
	case score >= 80:
		return "Sound — targeted hardening recommended"
	case score >= 65:
		return "Mixed — material gaps in at least one pillar"
	case score >= 50:
		return "Weak — significant work required before launch"
	default:
		return "Critical — system is not production-viable; rework required"
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func normalizeSystemInput(in SystemInput) SystemInput {
	if len(in.Services) == 0 && len(in.ServicesFlat) > 0 {
		for _, n := range in.ServicesFlat {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			in.Services = append(in.Services, ServiceDescriptor{Name: n})
		}
	}
	return in
}

func validateSystemForReview(in SystemInput) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Services) == 0 {
		return errors.New("at least one service is required for review")
	}
	return nil
}

func criticalityOf(c ServiceDescriptor) string {
	switch strings.ToLower(strings.TrimSpace(c.Criticality)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func sharedDataStores(in SystemInput) []string {
	owners := map[string][]string{}
	for _, svc := range in.Services {
		for _, ds := range svc.OwnsData {
			owners[ds] = append(owners[ds], svc.Name)
		}
	}
	out := []string{}
	for ds, owns := range owners {
		if len(owns) > 1 {
			out = append(out, ds)
		}
	}
	sort.Strings(out)
	return out
}

func mentionsAny(list []string, needle string) bool {
	n := strings.ToLower(needle)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), n) {
			return true
		}
	}
	return false
}

func anyShared(items []string, substr string) bool {
	for _, it := range items {
		if strings.Contains(it, substr) {
			return true
		}
	}
	return false
}

func anyConstraintMentions(constraints []string, needles ...string) bool {
	for _, c := range constraints {
		lc := strings.ToLower(c)
		for _, n := range needles {
			if strings.Contains(lc, n) {
				return true
			}
		}
	}
	return false
}

func diff(a, b []string) []string {
	bset := map[string]bool{}
	for _, x := range b {
		bset[x] = true
	}
	out := []string{}
	for _, x := range a {
		if !bset[x] {
			out = append(out, x)
		}
	}
	return out
}

func hasCrossServiceAsync(in SystemInput) bool {
	for _, svc := range in.Services {
		if len(svc.Async) > 0 {
			return true
		}
	}
	return false
}

func hasReplicatedCriticalServices(in SystemInput) bool {
	for _, svc := range in.Services {
		if criticalityOf(svc) == "high" && svc.Replicated {
			return true
		}
	}
	return false
}
