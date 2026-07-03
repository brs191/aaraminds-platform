// Package diagrams implements the service-layer logic for the
// generate_diagram_assets MCP tool.
//
// The service takes a structured architecture description and a target audience
// + diagram type, then produces three artifacts:
//
//   - Mermaid source text (renderable in markdown today)
//   - PlantUML source text (renderable in any PlantUML host)
//   - A draw.io prompt (the natural-language instruction a user can paste
//     into draw.io's AI feature, or copy into a tool that uses it)
//
// Supported diagram types: context, deployment, sequence, event_flow,
// service_boundary. The output is deterministic given the same input.
package diagrams

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for diagram assets.
type Input struct {
	SystemName      string           `json:"system_name"`
	Description     string           `json:"description,omitempty"`
	Audience        string           `json:"audience,omitempty"` // business | technical | executive | engineering
	DiagramType     string           `json:"diagram_type"`       // context | deployment | sequence | event_flow | service_boundary
	Services        []Service        `json:"services,omitempty"`
	Events          []Event          `json:"events,omitempty"`
	ExternalSystems []ExternalSystem `json:"external_systems,omitempty"`
}

// Service is one architectural component.
type Service struct {
	Name      string   `json:"name"`
	Type      string   `json:"type,omitempty"` // api | gateway | worker | function | datastore
	DependsOn []string `json:"depends_on,omitempty"`
	OwnsData  []string `json:"owns_data,omitempty"`
}

// Event is a domain event in the system.
type Event struct {
	Name      string   `json:"name"`
	Producer  string   `json:"producer"`
	Consumers []string `json:"consumers,omitempty"`
}

// ExternalSystem is anything outside the system boundary.
type ExternalSystem struct {
	Name      string `json:"name"`
	Direction string `json:"direction,omitempty"` // inbound | outbound | bi
}

// Output bundles all three asset forms.
type Output struct {
	SystemName   string   `json:"system_name"`
	DiagramType  string   `json:"diagram_type"`
	Audience     string   `json:"audience"`
	Mermaid      string   `json:"mermaid"`
	PlantUML     string   `json:"plantuml"`
	DrawIOPrompt string   `json:"drawio_prompt"`
	Notes        []string `json:"notes"`
}

// GeneratorService is the diagram assets service.
type GeneratorService struct{}

// NewService constructs a Service.
func NewService() *GeneratorService { return &GeneratorService{} }

// Generate validates input and produces the assets.
func (s *GeneratorService) Generate(in Input) (Output, error) {
	if err := validate(in); err != nil {
		return Output{}, err
	}
	dt := strings.ToLower(strings.TrimSpace(in.DiagramType))
	audience := strings.ToLower(strings.TrimSpace(in.Audience))
	if audience == "" {
		audience = "technical"
	}

	out := Output{
		SystemName:  in.SystemName,
		DiagramType: dt,
		Audience:    audience,
	}

	switch dt {
	case "context":
		out.Mermaid = mermaidContext(in)
		out.PlantUML = plantUMLContext(in)
	case "deployment":
		out.Mermaid = mermaidDeployment(in)
		out.PlantUML = plantUMLDeployment(in)
	case "sequence":
		out.Mermaid = mermaidSequence(in)
		out.PlantUML = plantUMLSequence(in)
	case "event_flow":
		out.Mermaid = mermaidEventFlow(in)
		out.PlantUML = plantUMLEventFlow(in)
	case "service_boundary":
		out.Mermaid = mermaidServiceBoundary(in)
		out.PlantUML = plantUMLServiceBoundary(in)
	}
	out.DrawIOPrompt = drawIOPrompt(in, dt, audience)
	out.Notes = audienceNotes(audience, dt)
	return out, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	switch strings.ToLower(strings.TrimSpace(in.DiagramType)) {
	case "context", "deployment", "sequence", "event_flow", "service_boundary":
		// ok
	case "":
		return errors.New("diagram_type is required")
	default:
		return fmt.Errorf("diagram_type %q is not supported (use context | deployment | sequence | event_flow | service_boundary)", in.DiagramType)
	}
	if (in.DiagramType == "sequence" || in.DiagramType == "event_flow") && len(in.Services) == 0 && len(in.Events) == 0 {
		// sequence/event_flow need something to plot
	}
	return nil
}

// ---- Mermaid renderers -----------------------------------------------------

func mermaidContext(in Input) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	fmt.Fprintf(&b, "    user((User)) --> system[%s]\n", in.SystemName)
	for _, ext := range in.ExternalSystems {
		dir := strings.ToLower(ext.Direction)
		switch dir {
		case "inbound":
			fmt.Fprintf(&b, "    %s[%s] --> system\n", normaliseID(ext.Name), ext.Name)
		case "outbound":
			fmt.Fprintf(&b, "    system --> %s[%s]\n", normaliseID(ext.Name), ext.Name)
		default:
			fmt.Fprintf(&b, "    system <--> %s[%s]\n", normaliseID(ext.Name), ext.Name)
		}
	}
	return b.String()
}

func mermaidDeployment(in Input) string {
	var b strings.Builder
	b.WriteString("flowchart TB\n")
	b.WriteString("    subgraph Azure[Azure Subscription]\n")
	b.WriteString("        subgraph CAE[Container Apps Environment]\n")
	sorted := sortedServices(in.Services)
	for _, s := range sorted {
		fmt.Fprintf(&b, "            %s[%s\\n%s]\n", normaliseID(s.Name), s.Name, displayType(s.Type))
	}
	b.WriteString("        end\n")
	if len(in.Services) > 0 {
		b.WriteString("        subgraph Data[Data Tier]\n")
		seen := map[string]bool{}
		for _, s := range sorted {
			for _, d := range s.OwnsData {
				if seen[d] {
					continue
				}
				seen[d] = true
				fmt.Fprintf(&b, "            %s[(\"%s\")]\n", normaliseID(d), d)
			}
		}
		b.WriteString("        end\n")
	}
	b.WriteString("    end\n")
	for _, s := range sorted {
		for _, d := range s.OwnsData {
			fmt.Fprintf(&b, "    %s --> %s\n", normaliseID(s.Name), normaliseID(d))
		}
	}
	return b.String()
}

func mermaidSequence(in Input) string {
	var b strings.Builder
	b.WriteString("sequenceDiagram\n")
	b.WriteString("    participant U as User\n")
	for _, s := range sortedServices(in.Services) {
		fmt.Fprintf(&b, "    participant %s as %s\n", normaliseID(s.Name), s.Name)
	}
	if len(in.Services) > 0 {
		first := sortedServices(in.Services)[0]
		fmt.Fprintf(&b, "    U->>%s: request\n", normaliseID(first.Name))
		for _, s := range sortedServices(in.Services) {
			for _, dep := range s.DependsOn {
				fmt.Fprintf(&b, "    %s->>%s: call\n", normaliseID(s.Name), normaliseID(dep))
				fmt.Fprintf(&b, "    %s-->>%s: response\n", normaliseID(dep), normaliseID(s.Name))
			}
		}
		fmt.Fprintf(&b, "    %s-->>U: response\n", normaliseID(first.Name))
	}
	return b.String()
}

func mermaidEventFlow(in Input) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	for _, ev := range sortedEvents(in.Events) {
		producer := normaliseID(ev.Producer)
		for _, c := range ev.Consumers {
			fmt.Fprintf(&b, "    %s[%s] -- \"%s\" --> %s[%s]\n", producer, ev.Producer, ev.Name, normaliseID(c), c)
		}
	}
	return b.String()
}

func mermaidServiceBoundary(in Input) string {
	var b strings.Builder
	b.WriteString("flowchart TB\n")
	for _, s := range sortedServices(in.Services) {
		fmt.Fprintf(&b, "    subgraph %s_b[%s — %s]\n", normaliseID(s.Name), s.Name, displayType(s.Type))
		for _, d := range s.OwnsData {
			fmt.Fprintf(&b, "        %s_%s[(\"%s\")]\n", normaliseID(s.Name), normaliseID(d), d)
		}
		b.WriteString("    end\n")
	}
	for _, s := range sortedServices(in.Services) {
		for _, dep := range s.DependsOn {
			fmt.Fprintf(&b, "    %s_b -.-> %s_b\n", normaliseID(s.Name), normaliseID(dep))
		}
	}
	return b.String()
}

// ---- PlantUML renderers ----------------------------------------------------

func plantUMLContext(in Input) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("!include <C4/C4_Context>\n")
	b.WriteString(fmt.Sprintf("Person(user, \"User\")\n"))
	b.WriteString(fmt.Sprintf("System(sys, \"%s\")\n", in.SystemName))
	for _, ext := range in.ExternalSystems {
		b.WriteString(fmt.Sprintf("System_Ext(%s, \"%s\")\n", normaliseID(ext.Name), ext.Name))
	}
	b.WriteString("Rel(user, sys, \"uses\")\n")
	for _, ext := range in.ExternalSystems {
		dir := strings.ToLower(ext.Direction)
		switch dir {
		case "inbound":
			b.WriteString(fmt.Sprintf("Rel(%s, sys, \"sends to\")\n", normaliseID(ext.Name)))
		case "outbound":
			b.WriteString(fmt.Sprintf("Rel(sys, %s, \"calls\")\n", normaliseID(ext.Name)))
		default:
			b.WriteString(fmt.Sprintf("Rel(sys, %s, \"integrates with\")\n", normaliseID(ext.Name)))
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func plantUMLDeployment(in Input) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("node \"Azure Subscription\" {\n")
	b.WriteString("  node \"Container Apps Environment\" {\n")
	for _, s := range sortedServices(in.Services) {
		fmt.Fprintf(&b, "    component \"%s\\n(%s)\" as %s\n", s.Name, displayType(s.Type), normaliseID(s.Name))
	}
	b.WriteString("  }\n")
	b.WriteString("  database \"Data Tier\" as data\n")
	b.WriteString("}\n")
	for _, s := range sortedServices(in.Services) {
		for _, d := range s.OwnsData {
			fmt.Fprintf(&b, "%s --> data : %s\n", normaliseID(s.Name), d)
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func plantUMLSequence(in Input) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("actor User as U\n")
	for _, s := range sortedServices(in.Services) {
		fmt.Fprintf(&b, "participant \"%s\" as %s\n", s.Name, normaliseID(s.Name))
	}
	if len(in.Services) > 0 {
		first := sortedServices(in.Services)[0]
		fmt.Fprintf(&b, "U -> %s: request\n", normaliseID(first.Name))
		for _, s := range sortedServices(in.Services) {
			for _, dep := range s.DependsOn {
				fmt.Fprintf(&b, "%s -> %s: call\n", normaliseID(s.Name), normaliseID(dep))
				fmt.Fprintf(&b, "%s --> %s: response\n", normaliseID(dep), normaliseID(s.Name))
			}
		}
		fmt.Fprintf(&b, "%s --> U: response\n", normaliseID(first.Name))
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func plantUMLEventFlow(in Input) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	for _, ev := range sortedEvents(in.Events) {
		for _, c := range ev.Consumers {
			fmt.Fprintf(&b, "%s -> %s : %s\n", normaliseID(ev.Producer), normaliseID(c), ev.Name)
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func plantUMLServiceBoundary(in Input) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	for _, s := range sortedServices(in.Services) {
		fmt.Fprintf(&b, "package \"%s (%s)\" {\n", s.Name, displayType(s.Type))
		for _, d := range s.OwnsData {
			fmt.Fprintf(&b, "  database \"%s\"\n", d)
		}
		b.WriteString("}\n")
	}
	for _, s := range sortedServices(in.Services) {
		for _, dep := range s.DependsOn {
			fmt.Fprintf(&b, "%s ..> %s\n", normaliseID(s.Name), normaliseID(dep))
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

// ---- draw.io prompt --------------------------------------------------------

func drawIOPrompt(in Input, dt, audience string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Create a %s diagram for the system %q targeted at the %s audience.\n\n", dt, in.SystemName, audience)
	if in.Description != "" {
		fmt.Fprintf(&b, "System description: %s\n\n", in.Description)
	}
	if len(in.Services) > 0 {
		b.WriteString("Components:\n")
		for _, s := range sortedServices(in.Services) {
			fmt.Fprintf(&b, "- %s (%s)", s.Name, displayType(s.Type))
			if len(s.DependsOn) > 0 {
				fmt.Fprintf(&b, ", depends on: %s", strings.Join(s.DependsOn, ", "))
			}
			if len(s.OwnsData) > 0 {
				fmt.Fprintf(&b, ", owns data: %s", strings.Join(s.OwnsData, ", "))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(in.Events) > 0 {
		b.WriteString("Events:\n")
		for _, ev := range sortedEvents(in.Events) {
			fmt.Fprintf(&b, "- %s produced by %s, consumed by: %s\n", ev.Name, ev.Producer, strings.Join(ev.Consumers, ", "))
		}
		b.WriteString("\n")
	}
	if len(in.ExternalSystems) > 0 {
		b.WriteString("External systems:\n")
		for _, ext := range in.ExternalSystems {
			fmt.Fprintf(&b, "- %s (%s)\n", ext.Name, defaultStr(ext.Direction, "bi"))
		}
	}
	b.WriteString("\nStyle: use C4-Model conventions where applicable. Keep components in a single boundary box for the system.\n")
	return b.String()
}

func audienceNotes(audience, dt string) []string {
	notes := []string{
		fmt.Sprintf("Generated for %s audience.", audience),
		"Mermaid renders inline in GitHub-flavored markdown.",
		"PlantUML can be rendered via any PlantUML host or extension.",
		"The draw.io prompt is text — paste into draw.io's AI generator, or feed to a manual diagram tool.",
	}
	switch audience {
	case "business", "executive":
		notes = append(notes, "Audience note: avoid implementation jargon in any manual edits; keep node labels human-readable.")
	case "engineering":
		notes = append(notes, "Audience note: it's acceptable (and helpful) to include resilience controls, scaling rules, and protocol details when manually refining.")
	}
	if dt == "sequence" {
		notes = append(notes, "Sequence diagrams scale poorly past ~10 participants; consider splitting into per-workflow diagrams.")
	}
	return notes
}

// ---- helpers ---------------------------------------------------------------

func normaliseID(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

func displayType(t string) string {
	if t == "" {
		return "service"
	}
	return t
}

func defaultStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func sortedServices(in []Service) []Service {
	out := make([]Service, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedEvents(in []Event) []Event {
	out := make([]Event, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Producer == out[j].Producer {
			return out[i].Name < out[j].Name
		}
		return out[i].Producer < out[j].Producer
	})
	return out
}
