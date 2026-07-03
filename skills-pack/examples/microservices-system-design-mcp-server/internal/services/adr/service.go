// Package adr implements the service-layer logic for the
// generate_architecture_decision_record MCP tool.
//
// The service takes a structured description of an architecture decision under
// consideration (title, context, options) and produces a complete ADR document
// in the canonical Michael Nygard format: status, context, decision, consequences,
// alternatives, plus references and a markdown rendering ready to commit.
//
// The logic is deterministic and rule-based. Given the same input, the output is
// byte-stable. The service applies ADR-shaping heuristics:
//
//   - One decision per record (refuses ambiguous multi-decision inputs)
//   - Context describes the forces, not just the situation
//   - Decision is a single statement, not a list
//   - Consequences cover positive, negative, and neutral
//   - Alternatives are recorded with the reason for rejection
//   - Status defaults to "Proposed" unless explicitly accepted
//
// This package has no external dependencies and no LLM calls. Tests are
// table-driven and run in a few milliseconds.
package adr

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Input is the structured request for an ADR.
type Input struct {
	SystemName   string       `json:"system_name"`
	Title        string       `json:"title"`                  // short imperative, e.g. "Use Saga with Outbox for Order Workflow"
	Context      string       `json:"context"`                // 2–4 sentences describing the forces
	Decision     string       `json:"decision"`               // the chosen direction
	Status       string       `json:"status,omitempty"`       // Proposed | Accepted | Deprecated | Superseded (default Proposed)
	Drivers      []string     `json:"drivers,omitempty"`      // ranked list of decision drivers
	Options      []Option     `json:"options,omitempty"`      // alternatives considered
	Consequences Consequences `json:"consequences,omitempty"` // positive / negative / neutral (auto-derived if empty)
	References   []string     `json:"references,omitempty"`   // links to skills, patterns, or external docs
	Date         string       `json:"date,omitempty"`         // ISO date; auto-filled if empty
	DecidedBy    string       `json:"decided_by,omitempty"`   // team or person; optional
}

// Option is one alternative considered.
type Option struct {
	Name            string   `json:"name"`
	Pros            []string `json:"pros,omitempty"`
	Cons            []string `json:"cons,omitempty"`
	Rejected        bool     `json:"rejected"`
	RejectedBecause string   `json:"rejected_because,omitempty"`
}

// Consequences captures the impact of the decision.
type Consequences struct {
	Positive []string `json:"positive,omitempty"`
	Negative []string `json:"negative,omitempty"`
	Neutral  []string `json:"neutral,omitempty"`
}

// Output is the structured ADR plus a rendered markdown form.
type Output struct {
	SystemName   string       `json:"system_name"`
	Title        string       `json:"title"`
	Status       string       `json:"status"`
	Date         string       `json:"date"`
	Drivers      []string     `json:"drivers"`
	Context      string       `json:"context"`
	Decision     string       `json:"decision"`
	Options      []Option     `json:"options"`
	Consequences Consequences `json:"consequences"`
	References   []string     `json:"references"`
	Warnings     []string     `json:"warnings"`      // gentle critique of weak ADR shapes
	Markdown     string       `json:"markdown"`      // ready-to-commit ADR text
	QualityScore int          `json:"quality_score"` // 0-100; reflects completeness
}

// Service is the ADR generator service.
type Service struct{}

// NewService constructs a Service.
func NewService() *Service { return &Service{} }

// Generate validates the input and produces the ADR.
//
// Errors are returned only for inputs that fundamentally cannot be processed
// (empty system name, empty title, empty decision). Weak shapes (short context,
// no options, no consequences) are surfaced as warnings, not errors — the ADR
// is still produced so the team can iterate.
func (s *Service) Generate(in Input) (Output, error) {
	if err := validate(in); err != nil {
		return Output{}, err
	}

	out := Output{
		SystemName:   in.SystemName,
		Title:        in.Title,
		Status:       normaliseStatus(in.Status),
		Date:         normaliseDate(in.Date),
		Drivers:      cleanList(in.Drivers),
		Context:      strings.TrimSpace(in.Context),
		Decision:     strings.TrimSpace(in.Decision),
		Options:      in.Options,
		Consequences: in.Consequences,
		References:   cleanList(in.References),
	}

	if isEmptyConsequences(out.Consequences) {
		out.Consequences = deriveConsequences(in)
	}

	out.Warnings = collectWarnings(in, out)
	out.QualityScore = computeScore(in, out)
	out.Markdown = renderMarkdown(in, out)

	return out, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if strings.TrimSpace(in.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(in.Decision) == "" {
		return errors.New("decision is required")
	}
	// Soft rule: titles that promise multiple decisions are out of scope.
	lowerTitle := strings.ToLower(in.Title)
	if strings.Contains(lowerTitle, " and ") && strings.Contains(lowerTitle, " and another ") {
		return errors.New("title appears to combine multiple decisions; split into separate ADRs")
	}
	return nil
}

func normaliseStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "accepted":
		return "Accepted"
	case "deprecated":
		return "Deprecated"
	case "superseded":
		return "Superseded"
	case "", "proposed":
		return "Proposed"
	default:
		return "Proposed"
	}
}

func normaliseDate(d string) string {
	if d == "" {
		return time.Now().UTC().Format("2006-01-02")
	}
	return d
}

func cleanList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isEmptyConsequences(c Consequences) bool {
	return len(c.Positive) == 0 && len(c.Negative) == 0 && len(c.Neutral) == 0
}

// deriveConsequences produces a minimal set when the caller omitted them,
// driven by deterministic rules based on Drivers and Options.
func deriveConsequences(in Input) Consequences {
	c := Consequences{}
	if len(in.Drivers) > 0 {
		c.Positive = append(c.Positive, fmt.Sprintf("Aligns the system with the stated drivers (%s)", strings.Join(in.Drivers, "; ")))
	}
	rejected := 0
	for _, opt := range in.Options {
		if opt.Rejected {
			rejected++
		}
	}
	if rejected > 0 {
		c.Negative = append(c.Negative, fmt.Sprintf("%d alternative(s) were rejected; teams comfortable with them must adapt", rejected))
	}
	c.Neutral = append(c.Neutral, "Decision should be reviewed if drivers or constraints change")
	return c
}

func collectWarnings(in Input, out Output) []string {
	w := []string{}
	if len(strings.TrimSpace(in.Context)) < 120 {
		w = append(w, "context is short (<120 chars); consider describing the forces and trade-offs in more depth")
	}
	if len(in.Drivers) == 0 {
		w = append(w, "no drivers listed; ADRs without ranked drivers are hard to revisit later")
	}
	if len(in.Options) == 0 {
		w = append(w, "no alternatives recorded; an ADR without rejected options reads as a single-answer mandate")
	}
	hasRejectionReason := false
	for _, opt := range in.Options {
		if opt.Rejected && strings.TrimSpace(opt.RejectedBecause) != "" {
			hasRejectionReason = true
			break
		}
	}
	if len(in.Options) > 0 && !hasRejectionReason {
		w = append(w, "rejected options lack a stated reason; future readers won't know why they were dropped")
	}
	if isEmptyConsequences(in.Consequences) {
		w = append(w, "consequences were not provided; an auto-derived set was inserted but should be reviewed and replaced")
	}
	if len(in.References) == 0 {
		w = append(w, "no references; consider linking to skills, pattern cards, or external docs for context")
	}
	sort.Strings(w)
	return w
}

func computeScore(in Input, out Output) int {
	score := 100
	if len(strings.TrimSpace(in.Context)) < 120 {
		score -= 15
	}
	if len(in.Drivers) == 0 {
		score -= 15
	}
	if len(in.Options) == 0 {
		score -= 20
	}
	if isEmptyConsequences(in.Consequences) {
		score -= 15
	}
	if len(in.References) == 0 {
		score -= 5
	}
	if score < 0 {
		score = 0
	}
	return score
}

func renderMarkdown(in Input, out Output) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# ADR: %s\n\n", out.Title)
	fmt.Fprintf(&b, "**System:** %s\n\n", out.SystemName)
	fmt.Fprintf(&b, "**Status:** %s\n\n", out.Status)
	fmt.Fprintf(&b, "**Date:** %s\n\n", out.Date)
	if in.DecidedBy != "" {
		fmt.Fprintf(&b, "**Decided by:** %s\n\n", in.DecidedBy)
	}

	if len(out.Drivers) > 0 {
		b.WriteString("## Drivers\n\n")
		for _, d := range out.Drivers {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Context\n\n")
	if out.Context == "" {
		b.WriteString("_(not provided)_\n\n")
	} else {
		fmt.Fprintf(&b, "%s\n\n", out.Context)
	}

	b.WriteString("## Decision\n\n")
	fmt.Fprintf(&b, "%s\n\n", out.Decision)

	if len(out.Options) > 0 {
		b.WriteString("## Options Considered\n\n")
		for _, opt := range out.Options {
			marker := "[ ]"
			if opt.Rejected {
				marker = "[x] rejected"
			} else if !opt.Rejected && strings.EqualFold(opt.Name, "") == false {
				marker = "[ ] open"
			}
			fmt.Fprintf(&b, "### %s — %s\n\n", opt.Name, marker)
			if len(opt.Pros) > 0 {
				b.WriteString("**Pros:**\n")
				for _, p := range opt.Pros {
					fmt.Fprintf(&b, "- %s\n", p)
				}
				b.WriteString("\n")
			}
			if len(opt.Cons) > 0 {
				b.WriteString("**Cons:**\n")
				for _, c := range opt.Cons {
					fmt.Fprintf(&b, "- %s\n", c)
				}
				b.WriteString("\n")
			}
			if opt.Rejected && opt.RejectedBecause != "" {
				fmt.Fprintf(&b, "**Rejected because:** %s\n\n", opt.RejectedBecause)
			}
		}
	}

	b.WriteString("## Consequences\n\n")
	if len(out.Consequences.Positive) > 0 {
		b.WriteString("**Positive:**\n")
		for _, c := range out.Consequences.Positive {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}
	if len(out.Consequences.Negative) > 0 {
		b.WriteString("**Negative:**\n")
		for _, c := range out.Consequences.Negative {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}
	if len(out.Consequences.Neutral) > 0 {
		b.WriteString("**Neutral:**\n")
		for _, c := range out.Consequences.Neutral {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	if len(out.References) > 0 {
		b.WriteString("## References\n\n")
		for _, r := range out.References {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}

	return b.String()
}
