// Package eventcontract implements the service-layer logic for the
// generate_event_contract MCP tool.
//
// The service takes a structured description of a domain event (name, producer,
// payload fields, expected consumers) and produces a CloudEvents-shaped contract
// document: schema, semantics, ordering and idempotency guarantees, the
// transport mapping (Azure Service Bus or Event Grid), and per-consumer
// subscription notes. It surfaces common modeling problems (event-as-command,
// missing correlation ID, sensitive-field exposure) as warnings.
//
// The logic is deterministic and rule-based.
package eventcontract

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for an event contract.
type Input struct {
	SystemName  string   `json:"system_name"`
	EventName   string   `json:"event_name"`          // past-tense, e.g. "OrderCreated"
	Producer    string   `json:"producer"`            // service that emits it
	Consumers   []string `json:"consumers"`           // services that subscribe
	Fields      []Field  `json:"fields"`              // payload schema
	Transport   string   `json:"transport,omitempty"` // service_bus | event_grid | event_hubs (default: service_bus)
	Ordering    string   `json:"ordering,omitempty"`  // none | per_aggregate | global (default: per_aggregate)
	Description string   `json:"description,omitempty"`
}

// Field is one payload field.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string | integer | boolean | object | array | date-time | uuid
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"` // PII / PHI / PCI
}

// Output is the event contract document.
type Output struct {
	SystemName     string           `json:"system_name"`
	EventName      string           `json:"event_name"`
	Producer       string           `json:"producer"`
	Consumers      []ConsumerNote   `json:"consumers"`
	Schema         Schema           `json:"schema"`
	Transport      TransportBinding `json:"transport"`
	Ordering       string           `json:"ordering"`
	IdempotencyKey string           `json:"idempotency_key"`
	Warnings       []string         `json:"warnings"`
	Markdown       string           `json:"markdown"`
	QualityScore   int              `json:"quality_score"`
}

// Schema is a CloudEvents-shaped representation of the event.
type Schema struct {
	SpecVersion string          `json:"specversion"`
	Type        string          `json:"type"`              // reverse-DNS-style, e.g. com.example.order.OrderCreated
	Source      string          `json:"source"`            // URI of the producer
	Subject     string          `json:"subject,omitempty"` // aggregate identifier
	DataSchema  string          `json:"dataschema"`
	DataFields  []FieldExpanded `json:"data"`
}

// FieldExpanded is the schema-level field with normalized properties.
type FieldExpanded struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	Description  string `json:"description,omitempty"`
	Sensitive    bool   `json:"sensitive,omitempty"`
	HandlingNote string `json:"handling_note,omitempty"`
}

// TransportBinding maps the event onto its Azure transport.
type TransportBinding struct {
	AzureService  string   `json:"azure_service"`
	TopicOrEntity string   `json:"topic_or_entity"`
	Subscriptions []string `json:"subscriptions"`
	DLQ           string   `json:"dlq"`
}

// ConsumerNote is per-consumer guidance.
type ConsumerNote struct {
	Service string `json:"service"`
	Notes   string `json:"notes"`
}

// GeneratorService is the event contract service.
type GeneratorService struct{}

// NewService constructs a Service.
func NewService() *GeneratorService { return &GeneratorService{} }

// Generate validates and produces the event contract.
func (s *GeneratorService) Generate(in Input) (Output, error) {
	if err := validate(in); err != nil {
		return Output{}, err
	}

	out := Output{
		SystemName: in.SystemName,
		EventName:  in.EventName,
		Producer:   in.Producer,
		Ordering:   normaliseOrdering(in.Ordering),
	}

	transport := normaliseTransport(in.Transport)
	out.Transport = TransportBinding{
		AzureService:  transportAzureService(transport),
		TopicOrEntity: topicName(in),
		Subscriptions: subscriptionList(in),
		DLQ:           dlqFor(transport, in),
	}

	out.Schema = buildSchema(in)
	out.IdempotencyKey = idempotencyKey(in)
	out.Consumers = consumerNotes(in)
	out.Warnings = collectWarnings(in)
	out.QualityScore = computeScore(in, out)
	out.Markdown = renderMarkdown(in, out)

	return out, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if strings.TrimSpace(in.EventName) == "" {
		return errors.New("event_name is required")
	}
	if strings.TrimSpace(in.Producer) == "" {
		return errors.New("producer is required")
	}
	if len(in.Fields) == 0 {
		return errors.New("at least one field is required")
	}
	for i, f := range in.Fields {
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("fields[%d].name is required", i)
		}
		if strings.TrimSpace(f.Type) == "" {
			return fmt.Errorf("fields[%d].type is required", i)
		}
	}
	return nil
}

func normaliseTransport(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "event_grid":
		return "event_grid"
	case "event_hubs":
		return "event_hubs"
	default:
		return "service_bus"
	}
}

func normaliseOrdering(o string) string {
	switch strings.ToLower(strings.TrimSpace(o)) {
	case "global":
		return "global"
	case "none":
		return "none"
	default:
		return "per_aggregate"
	}
}

func transportAzureService(t string) string {
	switch t {
	case "event_grid":
		return "Azure Event Grid"
	case "event_hubs":
		return "Azure Event Hubs"
	default:
		return "Azure Service Bus"
	}
}

func topicName(in Input) string {
	domain := strings.ToLower(strings.TrimSpace(in.Producer))
	domain = strings.ReplaceAll(domain, "-service", "")
	return fmt.Sprintf("%s.events", domain)
}

func subscriptionList(in Input) []string {
	out := make([]string, 0, len(in.Consumers))
	for _, c := range in.Consumers {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%s/%s", strings.ToLower(c), strings.ToLower(in.EventName)))
	}
	sort.Strings(out)
	return out
}

func dlqFor(transport string, in Input) string {
	switch transport {
	case "service_bus":
		return "$DeadLetterQueue (built-in); alert on depth >0"
	case "event_grid":
		return "Configure Event Grid dead-lettering to Blob Storage; alert on writes"
	case "event_hubs":
		return "Event Hubs has no native DLQ; consumers must implement DLQ at the processor"
	default:
		return "TBD"
	}
}

func buildSchema(in Input) Schema {
	domain := strings.ToLower(strings.TrimSpace(in.Producer))
	domain = strings.ReplaceAll(domain, "-service", "")
	expanded := make([]FieldExpanded, 0, len(in.Fields)+3)

	// Always-present envelope-ish fields (in addition to CloudEvents envelope keys).
	expanded = append(expanded,
		FieldExpanded{Name: "event_id", Type: "uuid", Required: true,
			Description: "Idempotency key for consumers. Stable per logical event delivery."},
		FieldExpanded{Name: "occurred_at", Type: "date-time", Required: true,
			Description: "Timestamp when the event happened in the producer."},
		FieldExpanded{Name: "correlation_id", Type: "uuid", Required: true,
			Description: "Cross-service correlation for tracing."},
	)

	for _, f := range in.Fields {
		fe := FieldExpanded{
			Name:        f.Name,
			Type:        f.Type,
			Required:    f.Required,
			Description: f.Description,
			Sensitive:   f.Sensitive,
		}
		if f.Sensitive {
			fe.HandlingNote = "PII / PHI / PCI — must be encrypted in transit; redact in logs and audit emitters."
		}
		expanded = append(expanded, fe)
	}

	return Schema{
		SpecVersion: "1.0",
		Type:        fmt.Sprintf("com.%s.%s.%s", strings.ToLower(in.SystemName), domain, in.EventName),
		Source:      fmt.Sprintf("urn:%s:%s", in.SystemName, in.Producer),
		Subject:     "<aggregate-id>",
		DataSchema:  fmt.Sprintf("schemas/%s.v1.json", in.EventName),
		DataFields:  expanded,
	}
}

func idempotencyKey(in Input) string {
	return "event_id — consumers must dedup on this key with a window ≥ broker retention."
}

func consumerNotes(in Input) []ConsumerNote {
	out := make([]ConsumerNote, 0, len(in.Consumers))
	for _, c := range in.Consumers {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		out = append(out, ConsumerNote{
			Service: c,
			Notes: "Subscribe via dedicated subscription. Implement idempotent consumer pattern keyed on event_id. " +
				"Add DLQ alerting and a poison-message playbook.",
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}

func collectWarnings(in Input) []string {
	w := []string{}
	lower := strings.ToLower(in.EventName)
	if !endsInPastTense(lower) {
		w = append(w, "event_name does not look past-tense; events describe past facts, not commands (e.g. OrderCreated, not CreateOrder)")
	}
	if strings.HasPrefix(lower, "send") || strings.HasPrefix(lower, "do") || strings.HasPrefix(lower, "make") {
		w = append(w, "event_name reads like a command; consider rephrasing as a past-tense fact")
	}
	if len(in.Consumers) == 0 {
		w = append(w, "no consumers listed; an event without subscribers is just a log — record at least one expected consumer or downgrade to internal-only event")
	}
	hasCorrelation := false
	for _, f := range in.Fields {
		if strings.EqualFold(f.Name, "correlation_id") {
			hasCorrelation = true
		}
	}
	if !hasCorrelation {
		w = append(w, "no correlation_id in fields; the schema adds one by default, but producers must populate it from their request context")
	}
	sensitiveCount := 0
	for _, f := range in.Fields {
		if f.Sensitive {
			sensitiveCount++
		}
	}
	if sensitiveCount > 0 {
		w = append(w, fmt.Sprintf("%d sensitive field(s) declared; ensure log redaction and audit emitter sanitization are in place at every consumer", sensitiveCount))
	}
	if in.Transport == "event_hubs" {
		w = append(w, "Event Hubs has no native DLQ; consumer-side DLQ handling is required")
	}
	sort.Strings(w)
	return w
}

func endsInPastTense(s string) bool {
	return strings.HasSuffix(s, "ed") ||
		strings.HasSuffix(s, "ned") || // confirmed, planned (past-tense forms)
		strings.HasSuffix(s, "te") // some irregular -ate verbs in noun form fail this; warning is best-effort
}

func computeScore(in Input, out Output) int {
	score := 100
	score -= 10 * len(out.Warnings)
	if score < 0 {
		score = 0
	}
	return score
}

func renderMarkdown(in Input, out Output) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Event Contract: %s\n\n", out.EventName)
	fmt.Fprintf(&b, "**System:** %s\n\n", out.SystemName)
	fmt.Fprintf(&b, "**Producer:** %s\n\n", out.Producer)
	fmt.Fprintf(&b, "**Type:** `%s`\n\n", out.Schema.Type)
	if in.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", in.Description)
	}

	b.WriteString("## Transport\n\n")
	fmt.Fprintf(&b, "- **Service:** %s\n", out.Transport.AzureService)
	fmt.Fprintf(&b, "- **Topic/entity:** `%s`\n", out.Transport.TopicOrEntity)
	fmt.Fprintf(&b, "- **DLQ:** %s\n", out.Transport.DLQ)
	fmt.Fprintf(&b, "- **Ordering:** %s\n\n", out.Ordering)

	b.WriteString("## Subscriptions\n\n")
	for _, sub := range out.Transport.Subscriptions {
		fmt.Fprintf(&b, "- `%s`\n", sub)
	}
	b.WriteString("\n")

	b.WriteString("## Schema (CloudEvents v1.0)\n\n")
	b.WriteString("| Field | Type | Required | Sensitive | Description |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, f := range out.Schema.DataFields {
		req := "no"
		if f.Required {
			req = "yes"
		}
		sens := ""
		if f.Sensitive {
			sens = "yes"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", f.Name, f.Type, req, sens, f.Description)
	}
	b.WriteString("\n")

	b.WriteString("## Idempotency\n\n")
	fmt.Fprintf(&b, "%s\n\n", out.IdempotencyKey)

	b.WriteString("## Consumers\n\n")
	for _, c := range out.Consumers {
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", c.Service, c.Notes)
	}

	return b.String()
}
