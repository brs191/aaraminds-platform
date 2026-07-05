package aapruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArtifactSections is the required-section registry for Markdown artifacts,
// per execution-package/artifact-schemas.md. An artifact is complete when
// every required section is present with non-empty content — never by
// reviewer judgment alone (BRD v2.1 AC-003).
var ArtifactSections = map[string][]string{
	"agent-blueprint.md": {
		"Business Problem", "Users & Stakeholders", "Expected Outcomes",
		"Autonomy Level & Justification", "Workflow Overview", "Tools",
		"Data Domains", "Identity", "Security & Approval Boundaries",
		"Evaluation Approach", "Operations & Ownership", "Non-Goals",
	},
	"system-prompt.md": {
		"Role & Objective", "Evidence & Citation Rules", "Prohibited Behaviors",
		"Output Structure", "Escalation Rules",
	},
	"workflow-design.md": {
		"Trigger & Inputs", "Step Graph", "Approval Points",
		"Failure Handling per Step", "Completion Criteria",
	},
	"mcp-tool-contracts.md": {
		"Contract Index",
	},
	"agent-identity-spec.md": {
		"Principal", "Credential Pattern", "Scopes", "Conditional Access",
		"Lifecycle & Owner",
	},
	"data-and-evidence-contract.md": {
		"Domain Table", "Evidence Rules", "Staleness and Conflict Notes",
	},
	"security-governance-checklist.md": {
		"ASI01 Planning & Goal Manipulation", "ASI02 Tool Misuse",
		"ASI03 Identity & Privilege Abuse", "ASI04 Agentic Supply Chain",
		"ASI05 Unsafe Code Execution", "ASI06 Memory Poisoning",
		"ASI07 Inter-Agent Communication", "ASI08 Cascading Failures",
		"ASI09 Human-Agent Trust Exploitation", "ASI10 Rogue Agents",
		"RBAC Summary", "Data Classification Summary", "Audit Obligations",
		"Kill-Switch Path",
	},
	"evaluation-plan.md": {
		"Golden Tests", "Tool Accuracy", "Retrieval, Evidence, and Citations",
		"Safety and Prompt Injection", "Latency", "Cost", "Regression",
	},
	"compliance-evidence-map.md": {
		"AI Act Role Assessment", "ISO 42001 Registry Fields",
		"NIST AI RMF Function Mapping", "Open Compliance Questions",
	},
	"implementation-backlog.md": {
		"Epics", "Stories", "Dependencies", "Estimate Class",
	},
	"risk-register.md": {
		"Risk Table",
	},
}

// SectionReport lists missing or empty sections for one artifact.
type SectionReport struct {
	Artifact string   `json:"artifact"`
	Missing  []string `json:"missing"`
	Empty    []string `json:"empty"`
}

func (r SectionReport) OK() bool { return len(r.Missing) == 0 && len(r.Empty) == 0 }

// ValidateArtifactSections checks one Markdown file against a required
// section list. A section is a "## " heading; its content is everything up
// to the next heading of any level.
func ValidateArtifactSections(path string, required []string) (SectionReport, error) {
	report := SectionReport{Artifact: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		return report, fmt.Errorf("read artifact %s: %w", path, err)
	}
	content := sectionContents(string(raw))
	for _, name := range required {
		body, found := content[name]
		if !found {
			report.Missing = append(report.Missing, name)
			continue
		}
		if strings.TrimSpace(body) == "" {
			report.Empty = append(report.Empty, name)
		}
	}
	return report, nil
}

// ValidateArtifactDir validates every registered Markdown artifact present in
// dir and reports artifacts that are registered but absent as missing files.
func ValidateArtifactDir(dir string) ([]SectionReport, error) {
	reports := make([]SectionReport, 0, len(ArtifactSections))
	names := sortedArtifactNames()
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			reports = append(reports, SectionReport{Artifact: path, Missing: []string{"<file missing>"}})
			continue
		}
		report, err := ValidateArtifactSections(path, ArtifactSections[name])
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func sortedArtifactNames() []string {
	names := make([]string, 0, len(ArtifactSections))
	for name := range ArtifactSections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sectionContents(markdown string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(markdown, "\n")
	var current string
	var body strings.Builder
	flush := func() {
		if current != "" {
			sections[current] = body.String()
		}
		body.Reset()
	}
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Fenced code blocks are opaque: a "## " line inside a fence is
		// content, not a section boundary. Without this, a code sample could
		// silently truncate a section the readiness engine then trusts.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			if current != "" {
				body.WriteString(line)
				body.WriteString("\n")
			}
			continue
		}
		if inFence {
			if current != "" {
				body.WriteString(line)
				body.WriteString("\n")
			}
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "### ") {
			// any other heading level ends the current section body
			flush()
			current = ""
			continue
		}
		if current != "" {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}
	flush()
	return sections
}
