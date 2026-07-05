package aapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRubricValid(t *testing.T) {
	root := repoRootForTest(t)
	rubric, err := LoadRubric(root)
	if err != nil {
		t.Fatalf("LoadRubric: %v", err)
	}
	if rubric.RubricVersion == "" || len(rubric.Areas) != 9 {
		t.Fatalf("unexpected rubric: version %q, %d areas", rubric.RubricVersion, len(rubric.Areas))
	}
	if rubric.Thresholds.Pass <= rubric.Thresholds.Defer {
		t.Fatalf("pass threshold must exceed defer: %+v", rubric.Thresholds)
	}
}

func TestLoadRubricRejectsUnknownCheck(t *testing.T) {
	root := repoRootForTest(t)
	raw, err := os.ReadFile(filepath.Join(root, "governance", "readiness-rubric.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(raw), "intake-valid", "no-such-check", 1)
	tmpRoot := t.TempDir()
	mustMkdirAll(t, filepath.Join(tmpRoot, "governance"))
	mustMkdirAll(t, filepath.Join(tmpRoot, "schemas"))
	copyTestFile(t, filepath.Join(root, "schemas", "readiness-rubric.schema.json"), filepath.Join(tmpRoot, "schemas", "readiness-rubric.schema.json"))
	if err := os.WriteFile(filepath.Join(tmpRoot, "governance", "readiness-rubric.yaml"), []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRubric(tmpRoot); err == nil || !strings.Contains(err.Error(), "unknown check") {
		t.Fatalf("expected unknown-check failure, got: %v", err)
	}
}

func TestLoadRubricRejectsBadWeights(t *testing.T) {
	root := repoRootForTest(t)
	raw, err := os.ReadFile(filepath.Join(root, "governance", "readiness-rubric.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(raw), "weight: 10", "weight: 11", 1)
	tmpRoot := t.TempDir()
	mustMkdirAll(t, filepath.Join(tmpRoot, "governance"))
	mustMkdirAll(t, filepath.Join(tmpRoot, "schemas"))
	copyTestFile(t, filepath.Join(root, "schemas", "readiness-rubric.schema.json"), filepath.Join(tmpRoot, "schemas", "readiness-rubric.schema.json"))
	if err := os.WriteFile(filepath.Join(tmpRoot, "governance", "readiness-rubric.yaml"), []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRubric(tmpRoot); err == nil || !strings.Contains(err.Error(), "sum") {
		t.Fatalf("expected weight-sum failure, got: %v", err)
	}
}

// TestReadinessBAAgentHonestVerdict runs the full engine against the real BA
// agent scaffold and asserts the engine is honest about its own reference
// agent: human-work TODOs (identity, ASI review, compliance, eval runs) must
// fail until actually resolved, harness-backed gates must pass, and the
// verdict must follow the rubric arithmetic — never sentiment in either
// direction.
func TestReadinessBAAgentHonestVerdict(t *testing.T) {
	root := repoRootForTest(t)
	agentDir := filepath.Join(root, "agents", "aara-business-analyst")
	if _, err := os.Stat(filepath.Join(agentDir, "agent-intake.yaml")); err != nil {
		t.Skipf("BA agent scaffold not present: %v", err)
	}
	manifest := filepath.Join(root, "examples", "ba-agent.manifest.yaml")

	report, err := RunReadiness(root, agentDir, manifest)
	if err != nil {
		t.Fatalf("RunReadiness: %v", err)
	}
	if report.Score <= 0 || report.Score >= 100 {
		t.Fatalf("implausible score %.1f", report.Score)
	}
	// Verdict must follow the rubric, not vibes.
	rubric, err := LoadRubric(root)
	if err != nil {
		t.Fatal(err)
	}
	wantVerdict := "block"
	switch {
	case len(report.CriticalBlockers) > 0:
		wantVerdict = "block"
	case report.Score >= rubric.Thresholds.Pass:
		wantVerdict = "pass"
	case report.Score >= rubric.Thresholds.Defer:
		wantVerdict = "defer"
	}
	if report.Verdict != wantVerdict {
		t.Fatalf("verdict %s does not follow rubric arithmetic (score %.1f, %d blockers, want %s)",
			report.Verdict, report.Score, len(report.CriticalBlockers), wantVerdict)
	}
	// The manifest/intake agent_id linkage must hold (fixed 2026-07-05).
	if !checkPassed(report, "manifest-agent-match") {
		t.Fatal("manifest agent_id must match intake agent_id")
	}
	// TODO-driven human-work checks must keep failing until the underlying
	// artifacts are actually completed — scaffolded placeholders never pass.
	for _, mustFailWhileTODO := range []string{"identity-complete", "asi-checklist-complete", "compliance-complete"} {
		if checkPassed(report, mustFailWhileTODO) {
			raw, _ := os.ReadFile(filepath.Join(agentDir, "agent-identity-spec.json"))
			if strings.Contains(string(raw), "[TODO") {
				t.Errorf("%s passed while TODO markers remain", mustFailWhileTODO)
			}
		}
	}
	// No eval run has been recorded yet; the check must reflect that.
	if _, err := os.Stat(filepath.Join(agentDir, "eval-runs")); err != nil {
		if checkPassed(report, "eval-runs-present") {
			t.Error("eval-runs-present must fail with no recorded eval runs")
		}
	}
	// Harness-backed gates must pass — the proof harness is green, including
	// the prompt-injection and memory-citation gates.
	for _, mustPass := range []string{"proof-tool-denial", "proof-memory-isolation", "proof-audit-chain",
		"prompt-injection-gate", "memory-citation-gate", "contracts-lint", "identity-valid"} {
		if !checkPassed(report, mustPass) {
			t.Errorf("expected %s to pass", mustPass)
		}
	}
	// export-roundtrip passes only when a current attestation exists.
	attested := false
	if _, err := os.Stat(filepath.Join(agentDir, "export-verification.json")); err == nil {
		if digest, err := ContentDigest(agentDir); err == nil {
			var verification ExportVerification
			if _, err := loadStructuredFile(filepath.Join(agentDir, "export-verification.json"), &verification); err == nil {
				attested = verification.Identical && verification.ContentDigest == digest
			}
		}
	}
	if checkPassed(report, "export-roundtrip") != attested {
		t.Errorf("export-roundtrip result must mirror attestation state (attested=%v)", attested)
	}
}

func TestReadinessReportSchemaValid(t *testing.T) {
	root := repoRootForTest(t)
	agentDir := filepath.Join(root, "agents", "aara-business-analyst")
	if _, err := os.Stat(filepath.Join(agentDir, "agent-intake.yaml")); err != nil {
		t.Skipf("BA agent scaffold not present: %v", err)
	}
	report, err := RunReadiness(root, agentDir, filepath.Join(root, "examples", "ba-agent.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	if err := WriteReadinessReport(root, outDir, report); err != nil {
		t.Fatalf("WriteReadinessReport (includes schema self-check): %v", err)
	}
	md, err := os.ReadFile(filepath.Join(outDir, "readiness-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Verdict", "## Area Scores", "## Check Evidence", "never self-attestation"} {
		if !strings.Contains(string(md), want) {
			t.Errorf("markdown report missing %q", want)
		}
	}
}

func TestActivationGate(t *testing.T) {
	root := repoRootForTest(t)
	dir := t.TempDir()

	// Draft status: no report required.
	if err := ActivationGate(root, dir, "draft"); err != nil {
		t.Fatalf("draft must not require a report: %v", err)
	}
	// Active without a report: blocked.
	if err := ActivationGate(root, dir, "active"); err == nil {
		t.Fatal("active without a readiness report must be blocked")
	}

	// Active with a defer report: blocked.
	rubric, err := LoadRubric(root)
	if err != nil {
		t.Fatal(err)
	}
	report := minimalValidReport(rubric.RubricVersion, "defer", 76)
	if err := writeJSONArtifact(filepath.Join(dir, "readiness-report.json"), report); err != nil {
		t.Fatal(err)
	}
	if err := ActivationGate(root, dir, "active"); err == nil || !strings.Contains(err.Error(), "pass") {
		t.Fatalf("defer verdict must block activation, got: %v", err)
	}

	// Active with a pass report: allowed.
	report.Verdict = "pass"
	report.Score = 92
	if err := writeJSONArtifact(filepath.Join(dir, "readiness-report.json"), report); err != nil {
		t.Fatal(err)
	}
	if err := ActivationGate(root, dir, "active"); err != nil {
		t.Fatalf("pass verdict must allow activation: %v", err)
	}

	// Stale rubric version: blocked even with pass.
	report.RubricVersion = "0.0.1"
	if err := writeJSONArtifact(filepath.Join(dir, "readiness-report.json"), report); err != nil {
		t.Fatal(err)
	}
	if err := ActivationGate(root, dir, "active"); err == nil || !strings.Contains(err.Error(), "rubric") {
		t.Fatalf("stale rubric version must block activation, got: %v", err)
	}
}

func TestReadinessDeterministicModuloTimestamp(t *testing.T) {
	root := repoRootForTest(t)
	agentDir := filepath.Join(root, "agents", "aara-business-analyst")
	if _, err := os.Stat(filepath.Join(agentDir, "agent-intake.yaml")); err != nil {
		t.Skipf("BA agent scaffold not present: %v", err)
	}
	manifest := filepath.Join(root, "examples", "ba-agent.manifest.yaml")
	first, err := RunReadiness(root, agentDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunReadiness(root, agentDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if first.Score != second.Score || first.Verdict != second.Verdict {
		t.Fatalf("non-deterministic verdict: %.1f/%s vs %.1f/%s", first.Score, first.Verdict, second.Score, second.Verdict)
	}
}

func checkPassed(report ReadinessReport, id string) bool {
	for _, area := range report.Areas {
		for _, check := range area.Evidence {
			if check.Check == id {
				return check.Result == "pass"
			}
		}
	}
	return false
}

func minimalValidReport(rubricVersion, verdict string, score float64) ReadinessReport {
	areas := make([]ReportArea, 9)
	for i := range areas {
		areas[i] = ReportArea{
			Area: "area", Weight: 10, ChecksTotal: 1, ChecksPassed: 1, Score: 10,
			Evidence: []CheckResult{{Check: "c", Result: "pass", Mechanism: "schema-validation", EvidenceRef: "x"}},
		}
	}
	return ReadinessReport{
		AgentID: "test-agent", AgentVersion: "0.1.0", RubricVersion: rubricVersion,
		GeneratedAt: "2026-07-05T12:00:00Z",
		Owners:      ReportOwners{BusinessOwner: "b", TechnicalOwner: "t"},
		Autonomy:    ReportAutonomy{Level: 2, Justification: "test", RiskTier: "medium"},
		Areas:       areas, CriticalBlockers: []ReportBlocker{},
		Score: score, Verdict: verdict,
		ApprovalsReq: []ReportApproval{{Role: "business-owner"}},
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
