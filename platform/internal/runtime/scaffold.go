package aapruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ScaffoldAgent renders the agent artifact folder from a validated intake and
// its deterministic classification (Epic 3). Same intake in, same bytes out:
// no timestamps other than the intake's own submitted_at, no model calls.
//
// The generated folder is a starting point, not a finished design: sections
// an architect must complete carry [TODO] markers. Section presence is
// self-checked after generation; a scaffold that fails its own section
// validation is a bug, not a warning.
func ScaffoldAgent(root, intakePath, outDir string, force bool) (string, []string, error) {
	intake, err := LoadIntake(root, intakePath)
	if err != nil {
		return "", nil, err
	}
	classification, err := ClassifyAgent(intake)
	if err != nil {
		return "", nil, err
	}

	dir := filepath.Join(outDir, intake.AgentID)
	entries, readErr := os.ReadDir(dir)
	switch {
	case readErr == nil && len(entries) > 0 && !force:
		return "", nil, fmt.Errorf("scaffold target %s is not empty; use force to overwrite", dir)
	case readErr != nil && !os.IsNotExist(readErr):
		return "", nil, fmt.Errorf("inspect scaffold target %s: %w", dir, readErr)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create scaffold dir: %w", err)
	}
	if force {
		// Remove previously generated artifacts so a re-scaffold cannot leave
		// stale files behind (matters for the export round-trip, AC-009).
		// Only known generated names are removed; hand-added files survive.
		stale := append(sortedArtifactNames(), generatedJSONArtifacts...)
		stale = append(stale, generatedExtras...)
		for _, name := range stale {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
				return "", nil, fmt.Errorf("remove stale artifact %s: %w", name, err)
			}
		}
	}

	data := scaffoldData{
		Intake:         intake,
		Class:          classification,
		ContractStatus: contractStatus(root, intake.ProposedTools),
	}

	var files []string
	for _, name := range sortedArtifactNames() {
		tmpl, ok := artifactTemplates[name]
		if !ok {
			return "", nil, fmt.Errorf("no template registered for artifact %s", name)
		}
		rendered, err := renderTemplate(name, tmpl, data)
		if err != nil {
			return "", nil, err
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
			return "", nil, fmt.Errorf("write %s: %w", path, err)
		}
		files = append(files, path)
	}

	// Copy the intake into the agent directory: the folder must be
	// self-contained so the readiness engine (and later, export/import)
	// never depends on files outside it.
	intakeRaw, err := os.ReadFile(intakePath)
	if err != nil {
		return "", nil, fmt.Errorf("copy intake: %w", err)
	}
	intakeCopy := filepath.Join(dir, "agent-intake.yaml")
	if err := os.WriteFile(intakeCopy, intakeRaw, 0o644); err != nil {
		return "", nil, fmt.Errorf("write intake copy: %w", err)
	}
	files = append(files, intakeCopy)

	// Machine-validated JSON artifacts.
	identityPath := filepath.Join(dir, "agent-identity-spec.json")
	if err := writeJSONArtifact(identityPath, identitySpec(intake)); err != nil {
		return "", nil, err
	}
	if err := ValidateStructuredFile(identityPath, filepath.Join(root, "schemas", "agent-identity-spec.schema.json")); err != nil {
		return "", nil, fmt.Errorf("generated identity spec failed its own schema: %w", err)
	}
	files = append(files, identityPath)

	evidencePath := filepath.Join(dir, "data-evidence-contract.json")
	if err := writeJSONArtifact(evidencePath, evidenceContract(intake)); err != nil {
		return "", nil, err
	}
	if err := ValidateStructuredFile(evidencePath, filepath.Join(root, "schemas", "data-evidence-contract.schema.json")); err != nil {
		return "", nil, fmt.Errorf("generated evidence contract failed its own schema: %w", err)
	}
	files = append(files, evidencePath)

	classificationPath := filepath.Join(dir, "classification.json")
	if err := writeJSONArtifact(classificationPath, classification); err != nil {
		return "", nil, err
	}
	files = append(files, classificationPath)

	// Self-check: every registered Markdown artifact must pass its own
	// section requirements immediately after generation.
	reports, err := ValidateArtifactDir(dir)
	if err != nil {
		return "", nil, err
	}
	for _, report := range reports {
		if !report.OK() {
			return "", nil, fmt.Errorf("scaffold self-check failed for %s: missing %v empty %v",
				report.Artifact, report.Missing, report.Empty)
		}
	}
	return dir, files, nil
}

// generatedJSONArtifacts are the machine-validated JSON files the scaffold
// owns alongside the Markdown artifacts.
var generatedJSONArtifacts = []string{
	"agent-identity-spec.json",
	"data-evidence-contract.json",
	"classification.json",
}

// generatedExtras are additional scaffold-owned files cleaned on force.
var generatedExtras = []string{"agent-intake.yaml"}

type scaffoldData struct {
	Intake         AgentIntake
	Class          Classification
	ContractStatus map[string]string
}

func contractStatus(root string, tools []IntakeTool) map[string]string {
	status := make(map[string]string, len(tools))
	for _, tool := range tools {
		path := filepath.Join(root, "tool-contracts", tool.ToolName+".contract.yaml")
		if _, err := os.Stat(path); err == nil {
			status[tool.ToolName] = "exists"
		} else {
			status[tool.ToolName] = "missing — scaffold with aapctl contracts"
		}
	}
	return status
}

func renderTemplate(name, tmpl string, data scaffoldData) (string, error) {
	t, err := template.New(name).Funcs(template.FuncMap{
		"boundaryHint": boundaryHint,
		"add":          func(a, b int) int { return a + b },
	}).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return sb.String(), nil
}

func boundaryHint(writes bool) string {
	if writes {
		return "soft or hard (write action)"
	}
	return "none (read-only)"
}

func writeJSONArtifact(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func identitySpec(intake AgentIntake) map[string]any {
	scopes := make([]map[string]any, 0, len(intake.ProposedTools))
	for _, tool := range intake.ProposedTools {
		permission := "read"
		if tool.Writes {
			permission = "write"
		}
		scopes = append(scopes, map[string]any{
			"resource":      tool.ToolName,
			"permission":    permission,
			"environment":   "dev",
			"justification": tool.Description,
		})
	}
	if len(scopes) == 0 {
		// The identity schema requires at least one scope. An advise-only
		// agent with no tools still reads approved sources; emit a placeholder
		// that passes the schema and prompts the architect.
		scopes = append(scopes, map[string]any{
			"resource":      "[TODO: approved read-only source]",
			"permission":    "read",
			"environment":   "dev",
			"justification": "advisory agent reads approved sources only; no tools proposed at intake",
		})
	}
	return map[string]any{
		"agent_id":     intake.AgentID,
		"spec_version": "0.1.0",
		"principal": map[string]any{
			"principal_type":     "agent-identity",
			"idp":                "entra-id [VERIFY per implementation]",
			"distinct_from_user": true,
		},
		"credential": map[string]any{
			"pattern":                  "oauth2-federated-identity-credential",
			"shared_secrets_forbidden": true,
			"max_lifetime_hours":       24,
			"local_dev_fallback":       "isolated dev credential; never shared production credentials",
		},
		"scopes": scopes,
		"lifecycle": map[string]any{
			"provisioning": "[TODO: provisioning process per IdP]",
			"rotation":     "[TODO: rotation cadence]",
			"retirement":   "[TODO: retirement trigger and owner]",
			"owner":        intake.Owners.TechnicalOwner,
		},
	}
}

func evidenceContract(intake AgentIntake) map[string]any {
	domains := make([]map[string]any, 0, len(intake.DataDomains))
	for _, domain := range intake.DataDomains {
		domains = append(domains, map[string]any{
			"domain":               domain.Domain,
			"authoritative_source": domain.AuthoritativeSource,
			"record_type":          "read-only",
			"classification":       domain.Classification,
			"staleness_note":       "[TODO architect: confirm record type and staleness handling]",
		})
	}
	return map[string]any{
		"agent_id":         intake.AgentID,
		"contract_version": "0.1.0",
		"data_domains":     domains,
		"evidence_rules": map[string]any{
			"factual_claims_require_citation": true,
			"citation_format":                 "inline source reference (document id or query id)",
			"uncited_output_behavior":         "flag",
			"memory_write_citation_required":  true,
		},
	}
}
