// Package azuremap implements the service-layer logic for the
// map_patterns_to_azure_services MCP tool.
//
// The service takes a list of architecture patterns and deterministically
// maps each to the Azure services that implement it, with a rationale and
// alternatives, using a static, curated catalog. It also reports mapping
// findings (unknown patterns, deployment-target mismatch, missing
// cross-cutting patterns like secrets management or observability) and an
// overall mapping-coverage score.
//
// The logic is deterministic and rule-based. No LLM calls, no external
// dependencies. Tests are table-driven and run in a few milliseconds.
//
// The catalog reflects Azure service naming verified as of May 2026; see
// skills/mcp/00-ecosystem-facts.md for the freshness mechanism.
package azuremap

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for mapping patterns to Azure services.
type Input struct {
	SystemName       string   `json:"system_name"`
	Description      string   `json:"description,omitempty"`
	Patterns         []string `json:"patterns"`
	DeploymentTarget string   `json:"deployment_target,omitempty"` // aks | container_apps | app_service | functions | hybrid
	Constraints      []string `json:"constraints,omitempty"`
}

// Mapping is the structured output: the pattern-to-Azure mapping.
type Mapping struct {
	SystemName       string           `json:"system_name"`
	Mappings         []PatternMapping `json:"mappings"`
	UnmappedPatterns []string         `json:"unmapped_patterns"`
	MappingFindings  []Finding        `json:"mapping_findings"`
	Coverage         Coverage         `json:"coverage"`
	MappingScore     int              `json:"mapping_score"` // 0-100
	MappingRating    string           `json:"mapping_rating"`
	Summary          string           `json:"summary"`
}

// PatternMapping is one pattern mapped to Azure services.
type PatternMapping struct {
	Pattern       string         `json:"pattern"`
	AzureServices []AzureService `json:"azure_services"`
	Rationale     string         `json:"rationale"`
	Alternatives  []string       `json:"alternatives,omitempty"`
}

// AzureService is a named Azure service and the role it plays.
type AzureService struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Finding is a mapping-quality issue.
type Finding struct {
	Severity        string   `json:"severity"` // low | medium | high
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	PatternsRelated []string `json:"patterns_related,omitempty"`
	Recommendation  string   `json:"recommendation"`
}

// Coverage summarizes how many patterns were mapped.
type Coverage struct {
	MappedCount int `json:"mapped_count"`
	TotalCount  int `json:"total_count"`
}

// Mapper is the pattern-to-Azure service.
type Mapper struct{}

// NewService constructs a Mapper.
func NewService() *Mapper { return &Mapper{} }

type catalogEntry struct {
	services     []AzureService
	rationale    string
	alternatives []string
}

// catalog is the curated pattern -> Azure mapping. Keys are normalized
// (lowercase, underscores).
var catalog = map[string]catalogEntry{
	"api_gateway": {
		services:     []AzureService{{Name: "Azure API Management", Role: "Managed API gateway, throttling, and policy enforcement"}},
		rationale:    "API Management provides gateway, rate limiting, transformation, and a developer portal as a managed service.",
		alternatives: []string{"Application Gateway (L7 load balancing only)", "Azure Front Door (global edge)"},
	},
	"async_messaging": {
		services:     []AzureService{{Name: "Azure Service Bus", Role: "Durable queues and topics for async messaging"}},
		rationale:    "Service Bus provides ordered, transactional, dead-letter-capable messaging suited to commands and events.",
		alternatives: []string{"Azure Event Hubs (high-throughput streaming)", "Azure Storage Queues (simple, low-cost)"},
	},
	"event_streaming": {
		services:     []AzureService{{Name: "Azure Event Hubs", Role: "High-throughput event ingestion and streaming"}},
		rationale:    "Event Hubs handles millions of events/sec with partitioned consumers, ideal for telemetry and event sourcing feeds.",
		alternatives: []string{"Azure Service Bus (transactional messaging)", "Kafka on HDInsight (Kafka API compatibility)"},
	},
	"saga": {
		services:     []AzureService{{Name: "Azure Durable Functions", Role: "Orchestrator for long-running saga workflows"}},
		rationale:    "Durable Functions models compensating saga steps as orchestrations with built-in state and retry.",
		alternatives: []string{"Azure Logic Apps (low-code orchestration)", "Service Bus + custom orchestrator"},
	},
	"cqrs": {
		services:     []AzureService{{Name: "Azure SQL Database", Role: "Write model"}, {Name: "Azure Cosmos DB", Role: "Read-optimized query model"}},
		rationale:    "Separate write and read stores let each side scale and be modeled independently; Cosmos serves low-latency reads.",
		alternatives: []string{"Single Azure SQL with read replicas (simpler, weaker separation)"},
	},
	"event_sourcing": {
		services:     []AzureService{{Name: "Azure Cosmos DB", Role: "Append-only event store"}, {Name: "Azure Event Hubs", Role: "Event distribution"}},
		rationale:    "Cosmos provides an append-only, partitioned event log; Event Hubs distributes events to projections.",
		alternatives: []string{"Azure SQL append-only tables (transactional, lower scale)"},
	},
	"circuit_breaker": {
		services:     []AzureService{{Name: "Azure API Management", Role: "Gateway-level circuit breaker and retry policies"}},
		rationale:    "Resilience is primarily a code concern (Polly), but API Management enforces gateway-level breaker policies.",
		alternatives: []string{"Dapr resiliency policies (sidecar)", "Service mesh (Istio/Linkerd) on AKS"},
	},
	"service_discovery": {
		services:     []AzureService{{Name: "Azure Kubernetes Service", Role: "In-cluster DNS-based service discovery"}},
		rationale:    "AKS provides native service discovery via Kubernetes DNS; Container Apps provides it via the managed environment.",
		alternatives: []string{"Azure Container Apps built-in discovery", "Consul (self-managed)"},
	},
	"config_management": {
		services:     []AzureService{{Name: "Azure App Configuration", Role: "Centralized configuration and feature flags"}},
		rationale:    "App Configuration centralizes settings and feature flags with change history and labels per environment.",
		alternatives: []string{"Kubernetes ConfigMaps (cluster-scoped)"},
	},
	"secrets_management": {
		services:     []AzureService{{Name: "Azure Key Vault", Role: "Secret, key, and certificate storage with access policies"}},
		rationale:    "Key Vault stores secrets and certificates with managed-identity access and rotation, off the code path.",
		alternatives: []string{"Kubernetes Secrets (weaker isolation)"},
	},
	"relational_data": {
		services:     []AzureService{{Name: "Azure SQL Database", Role: "Managed relational store"}},
		rationale:    "Azure SQL provides a managed, HA relational database with point-in-time restore and elastic scale.",
		alternatives: []string{"Azure Database for PostgreSQL", "Azure Database for MySQL"},
	},
	"document_data": {
		services:     []AzureService{{Name: "Azure Cosmos DB", Role: "Globally distributed document store"}},
		rationale:    "Cosmos DB offers low-latency, multi-region document storage with tunable consistency.",
		alternatives: []string{"Azure SQL JSON columns (lower scale)"},
	},
	"cache": {
		services:     []AzureService{{Name: "Azure Cache for Redis", Role: "Low-latency distributed cache"}},
		rationale:    "Managed Redis offloads read pressure and holds ephemeral shared state with sub-millisecond latency.",
		alternatives: []string{"In-process cache (no cross-instance sharing)"},
	},
	"container_orchestration": {
		services:     []AzureService{{Name: "Azure Kubernetes Service", Role: "Managed Kubernetes orchestration"}},
		rationale:    "AKS runs containerized microservices with autoscaling, rolling updates, and a broad ecosystem.",
		alternatives: []string{"Azure Container Apps (serverless containers, less control)"},
	},
	"serverless_compute": {
		services:     []AzureService{{Name: "Azure Functions", Role: "Event-driven serverless compute"}},
		rationale:    "Functions scales to zero and bills per execution, ideal for spiky, event-driven workloads.",
		alternatives: []string{"Azure Container Apps jobs", "AKS with KEDA"},
	},
	"observability": {
		services:     []AzureService{{Name: "Azure Monitor", Role: "Metrics and alerts"}, {Name: "Application Insights", Role: "Distributed tracing and APM"}},
		rationale:    "Azure Monitor + Application Insights provide metrics, logs, traces, and alerting integrated across Azure.",
		alternatives: []string{"Prometheus + Grafana on AKS (self-managed)"},
	},
	"identity": {
		services:     []AzureService{{Name: "Microsoft Entra ID", Role: "Identity provider and OAuth2/OIDC authority"}},
		rationale:    "Entra ID issues and validates tokens for users and workloads and integrates with managed identities.",
		alternatives: []string{"Azure AD B2C (customer identity)"},
	},
	"blob_storage": {
		services:     []AzureService{{Name: "Azure Blob Storage", Role: "Object storage for unstructured data"}},
		rationale:    "Blob Storage provides tiered, durable object storage with lifecycle management and SAS access.",
		alternatives: []string{"Azure Files (SMB/NFS share)"},
	},
}

// deployment patterns that conflict with specific deployment targets.
var deploymentExpectations = map[string]string{
	"container_orchestration": "aks",
	"serverless_compute":      "functions",
}

// Map applies the catalog and returns a Mapping.
func (m *Mapper) Map(in Input) (Mapping, error) {
	if err := validate(in); err != nil {
		return Mapping{}, err
	}

	mappings := []PatternMapping{}
	unmapped := []string{}
	seen := map[string]bool{}
	for _, raw := range in.Patterns {
		key := normalize(raw)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		entry, ok := catalog[key]
		if !ok {
			unmapped = append(unmapped, raw)
			continue
		}
		mappings = append(mappings, PatternMapping{
			Pattern:       key,
			AzureServices: entry.services,
			Rationale:     entry.rationale,
			Alternatives:  entry.alternatives,
		})
	}

	sort.SliceStable(mappings, func(i, j int) bool { return mappings[i].Pattern < mappings[j].Pattern })
	sort.Strings(unmapped)

	findings := detectFindings(in, seen, unmapped)
	total := len(seen) + len(unmapped)
	score := computeScore(findings, len(mappings), total)
	rating := ratingForScore(score)

	return Mapping{
		SystemName:       in.SystemName,
		Mappings:         mappings,
		UnmappedPatterns: unmapped,
		MappingFindings:  findings,
		Coverage:         Coverage{MappedCount: len(mappings), TotalCount: total},
		MappingScore:     score,
		MappingRating:    rating,
		Summary:          composeSummary(in.SystemName, len(mappings), total, len(findings), score),
	}, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Patterns) == 0 {
		return errors.New("at least one pattern is required")
	}
	return nil
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func detectFindings(in Input, mappedKeys map[string]bool, unmapped []string) []Finding {
	findings := []Finding{}

	if len(unmapped) > 0 {
		findings = append(findings, Finding{
			Severity:        "low",
			Category:        "unknown_pattern",
			Description:     fmt.Sprintf("%d pattern(s) are not in the curated Azure catalog and were not mapped", len(unmapped)),
			PatternsRelated: append([]string{}, unmapped...),
			Recommendation:  "Verify the pattern name, or map it manually and propose a catalog addition with a verified Azure service.",
		})
	}

	target := normalize(in.DeploymentTarget)
	if target != "" {
		// Sort the expectation keys so multiple matching findings are emitted in deterministic order.
		expKeys := make([]string, 0, len(deploymentExpectations))
		for k := range deploymentExpectations {
			expKeys = append(expKeys, k)
		}
		sort.Strings(expKeys)
		for _, pattern := range expKeys {
			expected := deploymentExpectations[pattern]
			if mappedKeys[pattern] && target != expected && target != "hybrid" {
				findings = append(findings, Finding{
					Severity:        "medium",
					Category:        "deployment_mismatch",
					Description:     fmt.Sprintf("pattern %q typically targets %q but the declared deployment target is %q", pattern, expected, target),
					PatternsRelated: []string{pattern},
					Recommendation:  fmt.Sprintf("Either deploy %q on %s, or document why the alternative platform satisfies this pattern.", pattern, expected),
				})
			}
		}
	}

	if !mappedKeys["secrets_management"] {
		findings = append(findings, Finding{
			Severity:       "medium",
			Category:       "missing_secrets_management",
			Description:    "no secrets-management pattern is present; secrets are likely handled ad hoc",
			Recommendation: "Add the secrets_management pattern (Azure Key Vault) so credentials and certificates are off the code path.",
		})
	}
	if !mappedKeys["observability"] {
		findings = append(findings, Finding{
			Severity:       "medium",
			Category:       "missing_observability",
			Description:    "no observability pattern is present; the system cannot be operated to an SLO without it",
			Recommendation: "Add the observability pattern (Azure Monitor + Application Insights) before production.",
		})
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return rank(findings[i].Severity) > rank(findings[j].Severity)
		}
		return findings[i].Category < findings[j].Category
	})
	return findings
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

func computeScore(findings []Finding, mapped, total int) int {
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
	// Penalize unmapped coverage proportionally (up to 20 points).
	if total > 0 && mapped < total {
		score -= int(float64(20) * float64(total-mapped) / float64(total))
	}
	if score < 0 {
		score = 0
	}
	return score
}

func ratingForScore(score int) string {
	switch {
	case score >= 90:
		return "Pattern coverage is strong; mapping is production-ready"
	case score >= 75:
		return "Pattern coverage is sound; close targeted gaps"
	case score >= 60:
		return "Pattern coverage has material gaps; address before implementation"
	case score >= 40:
		return "Pattern coverage is weak; significant mapping work needed"
	default:
		return "Pattern coverage is insufficient; revisit the architecture patterns"
	}
}

func composeSummary(systemName string, mapped, total, findingCount, score int) string {
	return fmt.Sprintf(
		"Pattern-to-Azure mapping for %s: %d/%d patterns mapped, %d findings, mapping score %d/100.",
		systemName, mapped, total, findingCount, score,
	)
}
