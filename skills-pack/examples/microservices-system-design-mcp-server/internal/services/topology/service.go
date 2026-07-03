// Package topology implements the service-layer logic for the
// generate_deployment_topology MCP tool.
//
// The service takes a structured architecture description (services, data
// stores, deployment target, NFRs) and produces a deployment topology: per-
// service compute placement (replicas, scale rules, resource hints), per-data-
// store placement, ingress and network boundaries, recommended environment
// promotion path (dev/staging/prod), and gaps that block a clean deploy.
//
// The logic is deterministic and rule-based:
//
//   - Default platform: Container Apps unless deployment_target says otherwise
//   - Replica floor: 2 for high-criticality, 1 otherwise; 0 for batch workers
//   - Scale rules: HTTP concurrency for APIs, queue-depth for workers
//   - Per-tier resource hints (CPU/memory) that align with measured baselines
//   - Ingress: external for gateways/APIs, internal-only for workers
//   - Network segmentation by sensitivity (pci/phi data stores get isolated subnets)
//
// This package has no external dependencies and no LLM calls.
package topology

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for a deployment topology.
type Input struct {
	SystemName       string      `json:"system_name"`
	Description      string      `json:"description,omitempty"`
	DeploymentTarget string      `json:"deployment_target,omitempty"` // container_apps | aks | app_service | hybrid
	Services         []Service   `json:"services"`
	DataStores       []DataStore `json:"data_stores,omitempty"`
	Environments     []string    `json:"environments,omitempty"` // e.g., ["dev","staging","prod"]
	NFR              NFR         `json:"non_functional_requirements,omitempty"`
}

// Service describes a service to be deployed.
type Service struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`        // api | gateway | worker | function
	Criticality string `json:"criticality,omitempty"` // high | medium | low
	External    bool   `json:"external,omitempty"`    // true if reachable from public internet
	Stateful    bool   `json:"stateful,omitempty"`
}

// DataStore describes a backing store and its placement constraints.
type DataStore struct {
	Name           string `json:"name"`
	Kind           string `json:"kind,omitempty"`           // postgres | cosmos | redis | blob | servicebus
	Classification string `json:"classification,omitempty"` // pii | phi | pci | sensitive | public
}

// NFR captures the non-functional drivers that shape placement decisions.
type NFR struct {
	AvailabilityTarget string `json:"availability_target,omitempty"`
	MultiRegion        bool   `json:"multi_region,omitempty"`
	LatencyP99Ms       int    `json:"latency_p99_ms,omitempty"`
}

// Topology is the structured output.
type Topology struct {
	SystemName        string             `json:"system_name"`
	Platform          string             `json:"platform"`
	ServicePlacements []ServicePlacement `json:"service_placements"`
	DataPlacements    []DataPlacement    `json:"data_placements"`
	NetworkBoundaries []NetworkBoundary  `json:"network_boundaries"`
	EnvironmentPath   []string           `json:"environment_path"`
	Gaps              []string           `json:"gaps"`
	NextSteps         []string           `json:"next_steps"`
	Score             int                `json:"score"` // 0-100 readiness
	Summary           string             `json:"summary"`
}

// ServicePlacement describes where a service runs and how it scales.
type ServicePlacement struct {
	Service   string `json:"service"`
	Platform  string `json:"platform"`
	Replicas  string `json:"replicas"`   // e.g., "2-10"
	Ingress   string `json:"ingress"`    // external | internal | none
	ScaleRule string `json:"scale_rule"` // http_concurrency | queue_depth | none
	CPU       string `json:"cpu"`        // e.g., "0.5"
	MemoryGiB string `json:"memory_gib"` // e.g., "1"
	Notes     string `json:"notes,omitempty"`
}

// DataPlacement describes where data lives and its classification.
type DataPlacement struct {
	Name           string `json:"name"`
	AzureService   string `json:"azure_service"`
	Tier           string `json:"tier"`
	Classification string `json:"classification,omitempty"`
	Encryption     string `json:"encryption"`       // at_rest | at_rest_cmk | n/a
	Subnet         string `json:"subnet,omitempty"` // dedicated | shared
	BackupPolicy   string `json:"backup_policy"`
}

// NetworkBoundary describes a network segmentation rule.
type NetworkBoundary struct {
	Name          string   `json:"name"`
	Includes      []string `json:"includes"`
	Justification string   `json:"justification"`
}

// Service is the deployment topology generator.
type GeneratorService struct{}

// NewService constructs a Service.
func NewService() *GeneratorService { return &GeneratorService{} }

// Generate validates the input and produces the topology.
func (s *GeneratorService) Generate(in Input) (Topology, error) {
	if err := validate(in); err != nil {
		return Topology{}, err
	}

	platform := normalisePlatform(in.DeploymentTarget)

	out := Topology{
		SystemName: in.SystemName,
		Platform:   platform,
	}

	out.ServicePlacements = placeServices(in.Services, platform, in.NFR)
	out.DataPlacements = placeData(in.DataStores)
	out.NetworkBoundaries = boundariesFor(in)
	out.EnvironmentPath = environmentPath(in.Environments)
	out.Gaps = findGaps(in, out)
	out.NextSteps = nextSteps(in, out)
	out.Score = scoreTopology(in, out)
	out.Summary = summarise(in, out)

	return out, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Services) == 0 {
		return errors.New("at least one service is required")
	}
	for i, s := range in.Services {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("services[%d].name is required", i)
		}
	}
	return nil
}

func normalisePlatform(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "aks":
		return "aks"
	case "app_service":
		return "app_service"
	case "functions":
		return "functions"
	case "hybrid":
		return "hybrid"
	default:
		return "container_apps"
	}
}

func placeServices(services []Service, platform string, nfr NFR) []ServicePlacement {
	out := make([]ServicePlacement, 0, len(services))
	for _, s := range services {
		p := ServicePlacement{
			Service:  s.Name,
			Platform: platform,
		}

		switch strings.ToLower(s.Type) {
		case "worker":
			p.ScaleRule = "queue_depth"
			p.Ingress = "none"
		case "function":
			p.ScaleRule = "event_trigger"
			p.Ingress = "none"
			p.Platform = "functions"
		case "gateway":
			p.ScaleRule = "http_concurrency"
			p.Ingress = "external"
		default: // api or unspecified
			p.ScaleRule = "http_concurrency"
			if s.External {
				p.Ingress = "external"
			} else {
				p.Ingress = "internal"
			}
		}

		switch strings.ToLower(s.Criticality) {
		case "high":
			p.Replicas = "2-10"
			p.CPU = "0.5"
			p.MemoryGiB = "1"
		case "low":
			if strings.EqualFold(s.Type, "worker") {
				p.Replicas = "0-3"
			} else {
				p.Replicas = "1-3"
			}
			p.CPU = "0.25"
			p.MemoryGiB = "0.5"
		default:
			p.Replicas = "1-5"
			p.CPU = "0.5"
			p.MemoryGiB = "1"
		}

		if nfr.MultiRegion && strings.EqualFold(s.Criticality, "high") {
			p.Notes = "Deploy to two regions with Front Door-based failover"
		}
		if s.Stateful {
			p.Notes = strings.TrimSpace(p.Notes + " Stateful workload: requires persistent volumes or external state store; reconsider whether this should be stateless")
		}

		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}

func placeData(stores []DataStore) []DataPlacement {
	out := make([]DataPlacement, 0, len(stores))
	for _, ds := range stores {
		dp := DataPlacement{Name: ds.Name, Classification: ds.Classification}
		switch strings.ToLower(ds.Kind) {
		case "postgres":
			dp.AzureService = "Azure Database for PostgreSQL — Flexible Server"
			dp.Tier = "General Purpose, 2 vCore"
			dp.BackupPolicy = "Daily; PITR 7 days"
		case "cosmos":
			dp.AzureService = "Azure Cosmos DB"
			dp.Tier = "Serverless or 400 RU provisioned"
			dp.BackupPolicy = "Continuous, 30 days"
		case "redis":
			dp.AzureService = "Azure Cache for Redis"
			dp.Tier = "Standard C1"
			dp.BackupPolicy = "RDB snapshot, daily"
		case "blob":
			dp.AzureService = "Azure Blob Storage"
			dp.Tier = "Standard LRS (Hot)"
			dp.BackupPolicy = "Soft delete 30 days; lifecycle to Cool 90 days"
		case "servicebus":
			dp.AzureService = "Azure Service Bus"
			dp.Tier = "Standard"
			dp.BackupPolicy = "n/a (transactional messaging)"
		default:
			dp.AzureService = "TBD"
			dp.Tier = "TBD"
			dp.BackupPolicy = "TBD"
		}
		switch strings.ToLower(ds.Classification) {
		case "pci", "phi", "pii":
			dp.Encryption = "at_rest_cmk"
			dp.Subnet = "dedicated"
		case "sensitive":
			dp.Encryption = "at_rest_cmk"
			dp.Subnet = "shared"
		case "":
			dp.Encryption = "at_rest"
			dp.Subnet = "shared"
		default:
			dp.Encryption = "at_rest"
			dp.Subnet = "shared"
		}
		out = append(out, dp)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func boundariesFor(in Input) []NetworkBoundary {
	out := []NetworkBoundary{
		{
			Name:          "perimeter",
			Includes:      []string{"api-gateway", "external apis"},
			Justification: "Public-facing entry; behind WAF and OAuth/JWT validation.",
		},
		{
			Name:          "application",
			Includes:      []string{"internal services", "workers"},
			Justification: "East-west traffic; mTLS enforced via service mesh or Container Apps internal ingress.",
		},
	}

	for _, ds := range in.DataStores {
		switch strings.ToLower(ds.Classification) {
		case "pci", "phi":
			out = append(out, NetworkBoundary{
				Name:          fmt.Sprintf("isolation:%s", ds.Classification),
				Includes:      []string{ds.Name},
				Justification: fmt.Sprintf("Sensitive classification (%s) — dedicated subnet, deny-by-default network policy, audit on every access.", ds.Classification),
			})
		}
	}
	return out
}

func environmentPath(envs []string) []string {
	if len(envs) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		e = strings.TrimSpace(e)
		if e != "" {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	return out
}

func findGaps(in Input, out Topology) []string {
	gaps := []string{}
	if in.DeploymentTarget == "" {
		gaps = append(gaps, "deployment_target unspecified; defaulted to container_apps")
	}
	if len(in.DataStores) == 0 {
		gaps = append(gaps, "no data stores declared; review whether services truly have no persistent state")
	}
	if in.NFR.AvailabilityTarget == "" {
		gaps = append(gaps, "availability target unspecified; replica floors are best-effort defaults")
	}
	if !in.NFR.MultiRegion {
		highCrit := 0
		for _, s := range in.Services {
			if strings.EqualFold(s.Criticality, "high") {
				highCrit++
			}
		}
		if highCrit > 0 && in.NFR.AvailabilityTarget == "99.95" {
			gaps = append(gaps, "99.95% availability with single-region topology is unlikely; consider multi-region")
		}
	}
	sort.Strings(gaps)
	return gaps
}

func nextSteps(in Input, out Topology) []string {
	steps := []string{
		"Author Container Apps / AKS manifest scaffolds per service from the placements above",
		"Wire managed identities for each service and bind to Key Vault for secrets",
		"Configure health probes (/healthz) on every API and gateway placement",
	}
	if len(in.DataStores) > 0 {
		steps = append(steps, "Provision data tier per placement and verify backup policies in staging before prod")
	}
	for _, b := range out.NetworkBoundaries {
		if strings.HasPrefix(b.Name, "isolation:") {
			steps = append(steps, "Apply dedicated subnet and deny-by-default NSG/NetworkPolicy for sensitive-data isolation boundary")
			break
		}
	}
	if in.NFR.MultiRegion {
		steps = append(steps, "Configure Azure Front Door with regional origin pools and health-based failover")
	}
	return steps
}

func scoreTopology(in Input, out Topology) int {
	score := 100
	if len(out.Gaps) > 0 {
		score -= 5 * len(out.Gaps)
	}
	if in.DeploymentTarget == "" {
		score -= 5
	}
	if len(in.DataStores) == 0 {
		score -= 10
	}
	if in.NFR.AvailabilityTarget == "" {
		score -= 5
	}
	if score < 0 {
		score = 0
	}
	return score
}

func summarise(in Input, out Topology) string {
	return fmt.Sprintf(
		"Deployment topology for %s on %s: %d service placement(s), %d data placement(s), %d network boundary(ies), %d gap(s); environment path %s.",
		in.SystemName, out.Platform, len(out.ServicePlacements), len(out.DataPlacements),
		len(out.NetworkBoundaries), len(out.Gaps), strings.Join(out.EnvironmentPath, " → "))
}
