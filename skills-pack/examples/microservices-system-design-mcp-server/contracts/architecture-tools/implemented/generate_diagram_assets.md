# Tool Contract: generate_diagram_assets

## Status

Implemented

## Purpose

Generate three parallel diagram assets for a microservices architecture: Mermaid source (renderable inline in markdown), PlantUML source (renderable in any PlantUML host), and a draw.io-ready prompt. Diagram types supported: context, deployment, sequence, event_flow, service_boundary. Audience tailoring affects accompanying notes.

## Risk Level

Low. Generative/informational; produces text only.

## Approval Required

No.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string",
  "audience": "business | technical | executive | engineering (default: technical)",
  "diagram_type": "context | deployment | sequence | event_flow | service_boundary (required)",
  "services": [
    {
      "name": "string",
      "type": "api | gateway | worker | function | datastore",
      "depends_on": ["string", "..."],
      "owns_data": ["string", "..."]
    }
  ],
  "events": [
    {
      "name": "string",
      "producer": "string",
      "consumers": ["string", "..."]
    }
  ],
  "external_systems": [
    {
      "name": "string",
      "direction": "inbound | outbound | bi"
    }
  ]
}
```

## Output Schema

```json
{
  "system_name": "string",
  "diagram_type": "string",
  "audience": "string",
  "mermaid": "string — Mermaid source",
  "plantuml": "string — PlantUML source between @startuml/@enduml",
  "drawio_prompt": "string — natural language description for draw.io AI or manual editing",
  "notes": ["string", "..."]
}
```

## Diagram types

- **context**: system in the middle, users, external systems around it. C4-Context style for PlantUML.
- **deployment**: services placed in Azure Container Apps Environment with a data tier subgraph. Owned-data edges from each service.
- **sequence**: User → first service → dependent services → back. Useful for request flows.
- **event_flow**: producers → events → consumers. Best for understanding pub-sub fan-out.
- **service_boundary**: each service in its own subgraph with its owned data; dependency arrows between boundaries.

## Errors

- `system_name is required`
- `diagram_type is required`
- `diagram_type %q is not supported (use context | deployment | sequence | event_flow | service_boundary)`

## Notes

The Mermaid, PlantUML, and draw.io outputs are deterministic given the same input. Service and event ordering is alphabetical to keep the output stable; this also makes diff-friendly diagrams (a small input change produces a small output diff).

Audience tailoring affects the `notes` field only — the diagram source text is the same shape regardless of audience. The note guides the human reviewer on labeling and abstraction level when manually refining.

For diagrams beyond ~10 participants (sequence) or ~15 nodes (others), consider splitting into multiple diagrams; a note is added to remind the reviewer.
