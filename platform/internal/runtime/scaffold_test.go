package aapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scaffoldBAFixture(t *testing.T) (root, dir string) {
	t.Helper()
	root = repoRootForTest(t)
	intakePath := filepath.Join(root, "examples", "ba-agent.intake.yaml")
	if _, err := os.Stat(intakePath); err != nil {
		t.Skipf("example intake not present: %v", err)
	}
	out := t.TempDir()
	dir, files, err := ScaffoldAgent(root, intakePath, out, false)
	if err != nil {
		t.Fatalf("ScaffoldAgent: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no files generated")
	}
	return root, dir
}

func TestScaffoldGeneratesAllArtifacts(t *testing.T) {
	_, dir := scaffoldBAFixture(t)
	for name := range ArtifactSections {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing artifact %s: %v", name, err)
		}
	}
	for _, extra := range []string{"agent-identity-spec.json", "data-evidence-contract.json", "classification.json"} {
		if _, err := os.Stat(filepath.Join(dir, extra)); err != nil {
			t.Errorf("missing JSON artifact %s: %v", extra, err)
		}
	}
}

func TestScaffoldSectionsPass(t *testing.T) {
	_, dir := scaffoldBAFixture(t)
	reports, err := ValidateArtifactDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, report := range reports {
		if !report.OK() {
			t.Errorf("%s: missing %v empty %v", report.Artifact, report.Missing, report.Empty)
		}
	}
}

func TestScaffoldJSONValidatesAgainstSchemas(t *testing.T) {
	root, dir := scaffoldBAFixture(t)
	cases := map[string]string{
		"agent-identity-spec.json":    "agent-identity-spec.schema.json",
		"data-evidence-contract.json": "data-evidence-contract.schema.json",
	}
	for artifact, schema := range cases {
		if err := ValidateStructuredFile(filepath.Join(dir, artifact), filepath.Join(root, "schemas", schema)); err != nil {
			t.Errorf("%s: %v", artifact, err)
		}
	}
}

func TestScaffoldDeterministic(t *testing.T) {
	root, dir1 := scaffoldBAFixture(t)
	intakePath := filepath.Join(root, "examples", "ba-agent.intake.yaml")
	out2 := t.TempDir()
	dir2, _, err := ScaffoldAgent(root, intakePath, out2, false)
	if err != nil {
		t.Fatal(err)
	}
	for name := range ArtifactSections {
		a, err := os.ReadFile(filepath.Join(dir1, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir2, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Errorf("non-deterministic output for %s", name)
		}
	}
}

func TestScaffoldRefusesNonEmptyTargetWithoutForce(t *testing.T) {
	root, _ := scaffoldBAFixture(t)
	intakePath := filepath.Join(root, "examples", "ba-agent.intake.yaml")
	out := t.TempDir()
	if _, _, err := ScaffoldAgent(root, intakePath, out, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ScaffoldAgent(root, intakePath, out, false); err == nil {
		t.Fatal("expected refusal to overwrite non-empty target")
	}
	if _, _, err := ScaffoldAgent(root, intakePath, out, true); err != nil {
		t.Fatalf("force overwrite should succeed: %v", err)
	}
}

func TestSectionValidatorDetectsRemovedSection(t *testing.T) {
	_, dir := scaffoldBAFixture(t)
	path := filepath.Join(dir, "system-prompt.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(raw), "## Escalation Rules", "## Renamed Section", 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateArtifactSections(path, ArtifactSections["system-prompt.md"])
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() || len(report.Missing) != 1 || report.Missing[0] != "Escalation Rules" {
		t.Fatalf("expected missing Escalation Rules, got %+v", report)
	}
}

func TestSectionValidatorIgnoresHeadingsInsideCodeFences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "risk-register.md")
	content := "# Risk Register\n\n## Risk Table\n\nSome content.\n\n```text\n## This is code, not a heading\n# Neither is this\n```\n\nMore content after the fence.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateArtifactSections(path, ArtifactSections["risk-register.md"])
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("fenced pseudo-headings must not break sections: %+v", report)
	}
}

func TestScaffoldJSONDeterministic(t *testing.T) {
	root, dir1 := scaffoldBAFixture(t)
	intakePath := filepath.Join(root, "examples", "ba-agent.intake.yaml")
	out2 := t.TempDir()
	dir2, _, err := ScaffoldAgent(root, intakePath, out2, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range generatedJSONArtifacts {
		a, err := os.ReadFile(filepath.Join(dir1, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir2, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Errorf("non-deterministic JSON output for %s", name)
		}
	}
}

func TestScaffoldAdviseOnlyNoTools(t *testing.T) {
	root := repoRootForTest(t)
	content := `agent_id: advisory-test-agent
submitted_by: raja
submitted_at: "2026-07-05T10:00:00Z"
business_problem: Architecture reviews lack a consistent advisory framing and repeatable structure.
owners:
  business_owner: Raja Shekar Bollam
  technical_owner: Raja Shekar Bollam (acting engineering lead)
users:
  - Enterprise AI Architect
expected_outcomes:
  - Consistent advisory reviews
execution_intent: advise-only
proposed_tools: []
data_domains:
  - domain: architecture-docs
    authoritative_source: knowledge base
    classification: internal
risks:
  - Advisory output may be over-trusted
approval_needs: None; advisory only, no write actions.
classification_inputs:
  action_risk: low
  data_sensitivity: internal
  reversibility: reversible
  user_impact: low
  financial_impact: low
  production_impact: low
`
	fixtureDir := t.TempDir()
	intakePath := filepath.Join(fixtureDir, "intake.yaml")
	if err := os.WriteFile(intakePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	dir, _, err := ScaffoldAgent(root, intakePath, out, false)
	if err != nil {
		t.Fatalf("advise-only scaffold must succeed: %v", err)
	}
	// Identity spec must still pass its schema (minItems 1 on scopes).
	if err := ValidateStructuredFile(filepath.Join(dir, "agent-identity-spec.json"),
		filepath.Join(root, "schemas", "agent-identity-spec.schema.json")); err != nil {
		t.Fatalf("advise-only identity spec failed schema: %v", err)
	}
	reports, err := ValidateArtifactDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, report := range reports {
		if !report.OK() {
			t.Errorf("%s: missing %v empty %v", report.Artifact, report.Missing, report.Empty)
		}
	}
}

func TestSectionValidatorDetectsEmptySection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "risk-register.md")
	content := "# Risk Register\n\n## Risk Table\n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateArtifactSections(path, ArtifactSections["risk-register.md"])
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() || len(report.Empty) != 1 {
		t.Fatalf("expected empty Risk Table, got %+v", report)
	}
}
