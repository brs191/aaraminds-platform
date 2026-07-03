# Tool Contract: generate_deployment_topology

## Status

Implemented

## Purpose

Generate an Azure deployment topology for a microservices system: per-service compute placement (platform, replicas, scale rule, resource hints), per-data-store placement (Azure service, tier, encryption, subnet, backup policy), network segmentation boundaries, the environment promotion path, identified deployment gaps, recommended next steps, and a readiness score.

## Risk Level

Low. Informational/generative output; the tool does not provision resources.

## Approval Required

No. Generated topologies are recommendations a human review before applying.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string",
  "deployment_target": "container_apps | aks | app_service | functions | hybrid (default: container_apps)",
  "services": [
    {
      "name": "string (required)",
      "type": "api | gateway | worker | function (default: api)",
      "criticality": "high | medium | low (default: medium)",
      "external": "boolean — reachable from public internet",
      "stateful": "boolean — requires persistent local state"
    }
  ],
  "data_stores": [
    {
      "name": "string",
      "kind": "postgres | cosmos | redis | blob | servicebus",
      "classification": "pii | phi | pci | sensitive | public"
    }
  ],
  "environments": ["string", "..."],
  "non_functional_requirements": {
    "availability_target": "string — e.g. '99.9'",
    "multi_region": "boolean",
    "latency_p99_ms": "integer"
  }
}
```

## Output Schema

```json
{
  "system_name": "string",
  "platform": "string",
  "service_placements": [
    {
      "service": "string",
      "platform": "string",
      "replicas": "string — e.g. '2-10'",
      "ingress": "external | internal | none",
      "scale_rule": "http_concurrency | queue_depth | event_trigger | none",
      "cpu": "string — e.g. '0.5'",
      "memory_gib": "string — e.g. '1'",
      "notes": "string"
    }
  ],
  "data_placements": [
    {
      "name": "string",
      "azure_service": "string",
      "tier": "string",
      "classification": "string",
      "encryption": "at_rest | at_rest_cmk | n/a",
      "subnet": "dedicated | shared",
      "backup_policy": "string"
    }
  ],
  "network_boundaries": [
    {
      "name": "perimeter | application | isolation:<classification>",
      "includes": ["string", "..."],
      "justification": "string"
    }
  ],
  "environment_path": ["string", "..."],
  "gaps": ["string", "..."],
  "next_steps": ["string", "..."],
  "score": "integer 0-100",
  "summary": "string"
}
```

## Rules and defaults

- **Replica floor**: 2 minimum for high-criticality services; 1 for medium; workers may scale to 0.
- **Ingress**: gateways and external APIs get external ingress; internal APIs get internal-only; workers get no ingress.
- **Scale rule**: HTTP concurrency for APIs/gateways, queue depth for workers, event trigger for Functions.
- **Resource hints**: 0.5 vCPU / 1 GiB memory for medium-high criticality; 0.25 / 0.5 for low.
- **Sensitive data isolation**: PCI, PHI, or PII classifications get a dedicated subnet and CMK encryption; deny-by-default network policy added as a boundary.
- **Multi-region**: high-criticality services in multi-region NFR get a Front Door failover note.

## Errors

- `system_name is required`
- `at least one service is required`
- `services[N].name is required`

## Score

Starts at 100; deducts 5 per identified gap. Common gaps: unspecified deployment target, no data stores declared, no availability target, 99.95% NFR with single-region topology.
