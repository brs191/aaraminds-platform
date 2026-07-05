package aapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validIntakeYAML() string {
	return `agent_id: test-agent
submitted_by: raja
submitted_at: "2026-07-05T10:00:00Z"
business_problem: Requirements work is hand-assembled per engagement with no traceability.
owners:
  business_owner: Raja Shekar Bollam
  technical_owner: Raja Shekar Bollam (acting engineering lead)
users:
  - Enterprise AI Architect
expected_outcomes:
  - Reviewed requirements draft in under a day
execution_intent: draft-outputs
proposed_tools:
  - tool_name: get_project_context
    action_type: project_context_read
    writes: false
    description: Read engagement context.
  - tool_name: create_requirements_draft
    action_type: requirements_draft_create
    writes: true
    description: Create a requirements draft document.
data_domains:
  - domain: project-context
    authoritative_source: engagement repository
    classification: client-confidential
risks:
  - Draft quality depends on retrieval quality
approval_needs: Draft creation uses a soft approval boundary.
classification_inputs:
  action_risk: low
  data_sensitivity: client-confidential
  reversibility: reversible
  user_impact: medium
  financial_impact: low
  production_impact: low
`
}

// writeIntakeFixture writes an intake YAML into a temp dir and returns the
// repo root to use (schemas resolved from the real repo root).
func writeIntakeFixture(t *testing.T, content string) (root, path string) {
	t.Helper()
	root = repoRootForTest(t)
	dir := t.TempDir()
	path = filepath.Join(dir, "intake.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return root, path
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is platform/internal/runtime; repo root is three levels up
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

func TestLoadIntakeValid(t *testing.T) {
	root, path := writeIntakeFixture(t, validIntakeYAML())
	intake, err := LoadIntake(root, path)
	if err != nil {
		t.Fatalf("LoadIntake: %v", err)
	}
	if intake.AgentID != "test-agent" {
		t.Fatalf("agent_id = %q", intake.AgentID)
	}
	if len(intake.ProposedTools) != 2 || !intake.ProposedTools[1].Writes {
		t.Fatalf("proposed tools decoded incorrectly: %+v", intake.ProposedTools)
	}
}

func TestLoadIntakeMissingRequiredField(t *testing.T) {
	content := strings.Replace(validIntakeYAML(), "business_problem: Requirements work is hand-assembled per engagement with no traceability.\n", "", 1)
	root, path := writeIntakeFixture(t, content)
	if _, err := LoadIntake(root, path); err == nil {
		t.Fatal("expected schema validation failure for missing business_problem")
	}
}

func TestLoadIntakeBadEnum(t *testing.T) {
	content := strings.Replace(validIntakeYAML(), "execution_intent: draft-outputs", "execution_intent: full-send", 1)
	root, path := writeIntakeFixture(t, content)
	if _, err := LoadIntake(root, path); err == nil {
		t.Fatal("expected schema validation failure for bad execution_intent")
	}
}

func TestLoadIntakeIdenticalOwnersRejected(t *testing.T) {
	content := strings.Replace(validIntakeYAML(), "technical_owner: Raja Shekar Bollam (acting engineering lead)", "technical_owner: Raja Shekar Bollam", 1)
	root, path := writeIntakeFixture(t, content)
	_, err := LoadIntake(root, path)
	if err == nil || !strings.Contains(err.Error(), "identical") {
		t.Fatalf("expected identical-owners invariant failure, got: %v", err)
	}
}

func TestLoadIntakePIIDomainForcesSensitivity(t *testing.T) {
	content := strings.Replace(validIntakeYAML(), "classification: client-confidential", "classification: pii", 1)
	root, path := writeIntakeFixture(t, content)
	_, err := LoadIntake(root, path)
	if err == nil || !strings.Contains(err.Error(), "pii") {
		t.Fatalf("expected pii sensitivity invariant failure, got: %v", err)
	}
}

func TestLoadIntakeUnknownFieldRejected(t *testing.T) {
	content := validIntakeYAML() + "surprise_field: boo\n"
	root, path := writeIntakeFixture(t, content)
	if _, err := LoadIntake(root, path); err == nil {
		t.Fatal("expected failure for unknown top-level field")
	}
}

func TestLoadIntakeExampleFile(t *testing.T) {
	root := repoRootForTest(t)
	path := filepath.Join(root, "examples", "ba-agent.intake.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("example intake not present: %v", err)
	}
	if _, err := LoadIntake(root, path); err != nil {
		t.Fatalf("example intake must validate: %v", err)
	}
}
