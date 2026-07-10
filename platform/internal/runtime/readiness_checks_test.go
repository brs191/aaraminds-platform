package aapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// evalRunJSON builds a minimal schema-valid eval-run with the given result.
func evalRunJSON(result string) string {
	return `{
  "eval_id": "e1",
  "target_type": "agent",
  "target_id": "a",
  "target_version": "0.1.0",
  "benchmark_ref": "b",
  "threshold_profile": "t",
  "started_at": "2026-07-05T10:00:00Z",
  "completed_at": "2026-07-05T10:01:00Z",
  "overall_result": "` + result + `",
  "gate_results": [{"gate": "g", "result": "` + result + `", "score": null, "evidence_ref": "x"}]
}`
}

func evalCheck(t *testing.T, result string, hasRun bool) (bool, string) {
	t.Helper()
	root := repoRootForTest(t)
	agentDir := t.TempDir()
	if hasRun {
		mustMkdirAll(t, filepath.Join(agentDir, "eval-runs"))
		if err := os.WriteFile(filepath.Join(agentDir, "eval-runs", "e1.json"), []byte(evalRunJSON(result)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rc := &readinessContext{root: root, agentDir: agentDir}
	pass, _, fix := checkRegistry["eval-runs-pass"].run(rc)
	return pass, fix
}

func TestEvalRunsPassRequiresPassResult(t *testing.T) {
	if pass, _ := evalCheck(t, "pass", true); !pass {
		t.Error("a pass eval run must satisfy eval-runs-pass")
	}
	if pass, fix := evalCheck(t, "needs-review", true); pass {
		t.Errorf("needs-review must NOT satisfy eval-runs-pass; fix=%q", fix)
	}
	if pass, _ := evalCheck(t, "fail", true); pass {
		t.Error("fail must NOT satisfy eval-runs-pass")
	}
	if pass, _ := evalCheck(t, "pass", false); pass {
		t.Error("no eval run must NOT satisfy eval-runs-pass")
	}
}

func todoCheck(t *testing.T, blueprint string) bool {
	t.Helper()
	root := repoRootForTest(t)
	agentDir := t.TempDir()
	// Write a full set of section-complete artifacts, then mutate one.
	for _, name := range sortedArtifactNames() {
		if err := os.WriteFile(filepath.Join(agentDir, name), []byte(minimalSections(name)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent-blueprint.md"), []byte(blueprint), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := &readinessContext{root: root, agentDir: agentDir}
	pass, _, _ := checkRegistry["artifacts-todo-free"].run(rc)
	return pass
}

// minimalSections renders the required headings for an artifact so the file
// exists and is readable; content is a placeholder heading list.
func minimalSections(name string) string {
	var b strings.Builder
	b.WriteString("# " + name + "\n\n")
	for _, s := range ArtifactSections[name] {
		b.WriteString("## " + s + "\n\ncontent\n\n")
	}
	return b.String()
}

func TestArtifactsTodoFree(t *testing.T) {
	clean := "# Agent Blueprint\n\n## Business Problem\n\nresolved content\n"
	if !todoCheck(t, clean) {
		t.Error("clean artifacts must pass artifacts-todo-free")
	}
	withTodo := "# Agent Blueprint\n\n## Business Problem\n\n[TODO architect: fill this in]\n"
	if todoCheck(t, withTodo) {
		t.Error("a [TODO] marker must fail artifacts-todo-free")
	}
	withStatusTodo := "# Checklist\n\n## X\n\nStatus: TODO. pending.\n"
	if todoCheck(t, withStatusTodo) {
		t.Error("a 'Status: TODO' marker must fail artifacts-todo-free")
	}
}

func TestRubricV2LoadsAndReferencesNewChecks(t *testing.T) {
	root := repoRootForTest(t)
	rubric, err := LoadRubric(root)
	if err != nil {
		t.Fatalf("rubric must load: %v", err)
	}
	if rubric.RubricVersion != "0.2.0" {
		t.Errorf("expected rubric 0.2.0, got %s", rubric.RubricVersion)
	}
	// The tightened check ids must be present and the old id gone.
	all := map[string]bool{}
	for _, area := range rubric.Areas {
		for _, id := range area.Checks {
			all[id] = true
		}
	}
	if !all["eval-runs-pass"] || !all["artifacts-todo-free"] {
		t.Error("rubric must reference eval-runs-pass and artifacts-todo-free")
	}
	if all["eval-runs-present"] {
		t.Error("rubric must no longer reference the loose eval-runs-present check")
	}
}
