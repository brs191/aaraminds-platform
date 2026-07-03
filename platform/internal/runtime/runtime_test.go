package aapruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidManifestStarts(t *testing.T) {
	engine := newTestEngine(t)
	if err := engine.Start(testRunContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(engine.AuditEvents()) == 0 {
		t.Fatal("expected audit events")
	}
}

func TestMissingManifestFails(t *testing.T) {
	if _, err := NewEngine(repoRoot(t), "examples/missing.manifest.yaml", "tool-contracts"); err == nil {
		t.Fatal("expected missing manifest to fail")
	}
}

func TestOffManifestToolDeniedAndAudited(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("delete_production_record", map[string]any{"record_id": "x"}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied, got %+v", decision)
	}
	if !hasAuditEvent(engine.AuditEvents(), "tool_denied", decision.AuditEventID) {
		t.Fatalf("expected matching tool_denied audit event, got %+v", engine.AuditEvents())
	}
}

func TestRunScopedAuditEventsHaveRunID(t *testing.T) {
	engine := newStartedTestEngine(t)
	_ = engine.InvokeTool("get_project_context", map[string]any{"engagement_id": "eng-example-001"}, RuntimeInteractive, true)
	if !AuditRunEventsHaveRunID(engine.AuditEvents()) {
		t.Fatalf("run-scoped audit event is missing run_id: %+v", engine.AuditEvents())
	}
}

func TestSoftApprovalEscalatesToHardWhenUnattended(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("create_requirements_draft", map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
		"evidence_refs": []string{"source://example/test"},
	}, RuntimeUnattended, false)
	if decision.Outcome != "approval_required" {
		t.Fatalf("expected approval_required, got %+v", decision)
	}
	if decision.ApprovalBoundary != BoundaryHard {
		t.Fatalf("expected hard boundary after unattended escalation, got %q", decision.ApprovalBoundary)
	}
}

func TestPlatformReadyRawTelemetryRejected(t *testing.T) {
	root := repoRoot(t)
	manifest, err := LoadManifest(filepath.Join(root, "examples/ba-agent.manifest.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	manifest.Status = "platform-ready"
	manifest.Telemetry.PayloadMode = "raw-in-non-prod"

	dir := t.TempDir()
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "bad.manifest.yaml")
	if err := os.WriteFile(manifestPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngine(root, rel, "tool-contracts"); err == nil {
		t.Fatal("expected platform-ready raw telemetry manifest to fail")
	}
}

func TestTrueYAMLManifestValidates(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	writeFile(t, manifestPath, `
agent_id: aara-ba-agent
manifest_version: 1.0.0
owner: Raja
runtime: claude-agent-sdk
status: draft
allowed_skills:
  - skill_id: aara-ba-agent-core
    skill_version: existing-package-baseline
    source_path: skills-pack/agent-packages/aara-business-analyst/agent.md
allowed_tools:
  - tool_name: get_project_context
    contract_version: 1.0.0
    approval_boundary: none
memory:
  enabled: true
  scope: engagement
  allowed_classifications:
    - public
    - internal
    - client-confidential
  pii_allowed: false
approval_boundaries:
  default: hard
  blocked_actions_ref: governance/aap-blocked-actions.yaml
telemetry:
  otel_enabled: true
  cost_attribution: true
  payload_mode: hash-and-reference
evaluation_gate:
  required: false
  benchmark_ref: skills-pack/agent-packages/aara-business-analyst/eval-plan.md
  threshold_profile: skills-pack/agent-packages/aara-business-analyst/release-gate.json
`)
	rel, err := filepath.Rel(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngine(root, rel, "tool-contracts"); err != nil {
		t.Fatalf("expected true YAML manifest to validate: %v", err)
	}
}

func TestContractSchemaRejectsMissingRequiredField(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	contractsDir := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, manifestPath, `{
  "agent_id": "aara-ba-agent",
  "manifest_version": "1.0.0",
  "owner": "Raja",
  "runtime": "claude-agent-sdk",
  "status": "draft",
  "allowed_skills": [
    {
      "skill_id": "aara-ba-agent-core",
      "skill_version": "existing-package-baseline",
      "source_path": "skills-pack/agent-packages/aara-business-analyst/agent.md"
    }
  ],
  "allowed_tools": [
    {
      "tool_name": "broken_tool",
      "contract_version": "1.0.0",
      "approval_boundary": "none"
    }
  ],
  "memory": {
    "enabled": true,
    "scope": "engagement",
    "allowed_classifications": ["public", "internal", "client-confidential"],
    "pii_allowed": false
  },
  "approval_boundaries": {
    "default": "hard",
    "blocked_actions_ref": "governance/aap-blocked-actions.yaml"
  },
  "telemetry": {
    "otel_enabled": true,
    "cost_attribution": true,
    "payload_mode": "hash-and-reference"
  },
  "evaluation_gate": {
    "required": false,
    "benchmark_ref": "skills-pack/agent-packages/aara-business-analyst/eval-plan.md",
    "threshold_profile": "skills-pack/agent-packages/aara-business-analyst/release-gate.json"
  }
}`)
	writeFile(t, filepath.Join(contractsDir, "broken.contract.yaml"), `{
  "tool_name": "broken_tool",
  "contract_version": "1.0.0",
  "action_type": "broken_action",
  "purpose": "Intentionally broken contract.",
  "input_schema": { "type": "object" },
  "output_schema": { "type": "object" },
  "permissions_required": [],
  "approval_boundary": "none",
  "data_classification": { "input": "internal", "output": "internal" },
  "failure_modes": [
    {
      "code": "BROKEN",
      "meaning": "broken",
      "retryable": false,
      "safe_user_message": "broken"
    }
  ],
  "timeout_class": "interactive",
  "retry_policy": { "max_attempts": 0, "backoff": "none" },
  "audit_event_schema": { "type": "object" },
  "example_invocation": {}
}`)
	manifestRel, err := filepath.Rel(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	contractsRel, err := filepath.Rel(root, contractsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngine(root, manifestRel, contractsRel); err == nil {
		t.Fatal("expected schema validation to reject contract missing timeout_ms")
	}
}

func TestInvokeToolBeforeStartDeniedWithoutPanic(t *testing.T) {
	engine := newTestEngine(t)
	decision := engine.InvokeTool("get_project_context", map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "q",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" || decision.Reason != "run has not started" {
		t.Fatalf("expected pre-start denial, got %+v", decision)
	}
}

func TestToolPayloadSchemaViolationDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("create_requirements_draft", map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied for schema violation, got %+v", decision)
	}
	if decision.ApprovalBoundary != BoundaryBlocked {
		t.Fatalf("expected blocked boundary, got %+v", decision)
	}
}

func TestToolPayloadRejectsNonStringEngagement(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("get_project_context", map[string]any{
		"engagement_id": 123,
		"query":         "sample requirements discovery acceptance criteria",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied for non-string engagement_id, got %+v", decision)
	}
	if decision.Reason != "tool input does not match contract schema" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestToolEngagementMismatchDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("get_project_context", map[string]any{
		"engagement_id": "eng-other-001",
		"query":         "sample requirements discovery acceptance criteria",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied for engagement mismatch, got %+v", decision)
	}
	if decision.Reason != "tool payload engagement_id does not match active run" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestBlockedActionTypeDeniedEvenWhenManifestAllowsTool(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	contractsDir := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, manifestPath, `{
  "agent_id": "aara-ba-agent",
  "manifest_version": "1.0.0",
  "owner": "Raja",
  "runtime": "claude-agent-sdk",
  "status": "draft",
  "allowed_skills": [
    {
      "skill_id": "aara-ba-agent-core",
      "skill_version": "existing-package-baseline",
      "source_path": "skills-pack/agent-packages/aara-business-analyst/agent.md"
    }
  ],
  "allowed_tools": [
    {
      "tool_name": "delete_production_record",
      "contract_version": "1.0.0",
      "approval_boundary": "none"
    }
  ],
  "memory": {
    "enabled": true,
    "scope": "engagement",
    "allowed_classifications": ["public", "internal", "client-confidential"],
    "pii_allowed": false
  },
  "approval_boundaries": {
    "default": "hard",
    "blocked_actions_ref": "governance/aap-blocked-actions.yaml"
  },
  "telemetry": {
    "otel_enabled": true,
    "cost_attribution": true,
    "payload_mode": "hash-and-reference"
  },
  "evaluation_gate": {
    "required": false,
    "benchmark_ref": "skills-pack/agent-packages/aara-business-analyst/eval-plan.md",
    "threshold_profile": "skills-pack/agent-packages/aara-business-analyst/release-gate.json"
  }
}`)
	writeFile(t, filepath.Join(contractsDir, "delete.contract.yaml"), `{
  "tool_name": "delete_production_record",
  "contract_version": "1.0.0",
  "action_type": "production_delete",
  "purpose": "Delete a production record.",
  "input_schema": {
    "type": "object",
    "required": ["engagement_id", "record_id"],
    "properties": {
      "engagement_id": { "type": "string", "minLength": 1 },
      "record_id": { "type": "string", "minLength": 1 }
    },
    "additionalProperties": false
  },
  "output_schema": { "type": "object" },
  "permissions_required": ["production:record:delete"],
  "approval_boundary": "none",
  "data_classification": { "input": "client-confidential", "output": "client-confidential" },
  "failure_modes": [
    {
      "code": "BLOCKED",
      "meaning": "Production deletion is blocked.",
      "retryable": false,
      "safe_user_message": "This action is blocked."
    }
  ],
  "timeout_ms": 10000,
  "timeout_class": "interactive",
  "retry_policy": { "max_attempts": 0, "backoff": "none" },
  "audit_event_schema": {
    "type": "object",
    "required": ["run_id", "tool_name", "engagement_id", "input_hash", "outcome", "timestamp"],
    "properties": {
      "run_id": { "type": "string", "minLength": 1 },
      "tool_name": { "type": "string", "const": "delete_production_record" },
      "engagement_id": { "type": "string", "minLength": 1 },
      "input_hash": { "type": "string", "minLength": 1 },
      "outcome": { "type": "string", "enum": ["success", "denied", "approval_required"] },
      "timestamp": { "type": "string", "format": "date-time" },
      "reason": { "type": "string" },
      "action_type": { "type": "string" }
    },
    "additionalProperties": false
  },
  "example_invocation": {
    "engagement_id": "eng-example-001",
    "record_id": "record-001"
  }
}`)
	manifestRel, err := filepath.Rel(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	contractsRel, err := filepath.Rel(root, contractsDir)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(root, manifestRel, contractsRel)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if err := engine.Start(testRunContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	decision := engine.InvokeTool("delete_production_record", map[string]any{
		"engagement_id": "eng-example-001",
		"record_id":     "record-001",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied blocked action, got %+v", decision)
	}
	if decision.Reason != "tool action type is blocked" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestMemoryIsolation(t *testing.T) {
	engine := newStartedTestEngine(t)
	ctx := testRunContext()
	err := engine.WriteMemory(MemoryRecord{
		MemoryID:       "mem-test-001",
		AgentID:        ctx.AgentID,
		EngagementID:   ctx.EngagementID,
		Classification: "client-confidential",
		ContentRef:     "memory://eng-example-001/mem-test-001",
		ContentHash:    "hash-test",
		SourceCitation: SourceCitation{
			RunID:     ctx.RunID,
			TraceID:   "trace-" + ctx.RunID,
			SpanID:    "span-test",
			SourceRef: "source://example/test",
		},
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("write memory: %v", err)
	}
	if got := len(engine.QueryMemory("eng-other-001")); got != 0 {
		t.Fatalf("expected no cross-engagement memory, got %d", got)
	}
	if got := len(engine.QueryMemory(ctx.EngagementID)); got != 1 {
		t.Fatalf("expected engagement memory, got %d", got)
	}
}

func TestMemorySourceCitationMustMatchRun(t *testing.T) {
	engine := newStartedTestEngine(t)
	ctx := testRunContext()
	err := engine.WriteMemory(MemoryRecord{
		MemoryID:       "mem-test-002",
		AgentID:        ctx.AgentID,
		EngagementID:   ctx.EngagementID,
		Classification: "client-confidential",
		ContentRef:     "memory://eng-example-001/mem-test-002",
		ContentHash:    "hash-test",
		SourceCitation: SourceCitation{
			RunID:     "run-other-001",
			TraceID:   "trace-run-other-001",
			SpanID:    "span-test",
			SourceRef: "source://example/test",
		},
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Status:    "active",
	})
	if err == nil {
		t.Fatal("expected source citation run mismatch to fail")
	}
}

func TestPhase1ProofPasses(t *testing.T) {
	report, err := RunPhase1Proof(repoRoot(t))
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	if !report.ValidManifestStarted ||
		!report.AllowedToolExecuted ||
		!report.OffManifestToolDenied ||
		!report.DenialAuditLogged ||
		!report.SoftUnattendedEscalated ||
		report.MemoryLeakageReturned != 0 ||
		!report.RunScopedAuditsHaveRunID ||
		report.TraceSpanCount == 0 {
		t.Fatalf("proof did not pass: %+v", report)
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine(repoRoot(t), "examples/ba-agent.manifest.yaml", "tool-contracts")
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return engine
}

func newStartedTestEngine(t *testing.T) *Engine {
	t.Helper()
	engine := newTestEngine(t)
	if err := engine.Start(testRunContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	return engine
}

func testRunContext() RunContext {
	return RunContext{
		RunID:           "run-test-001",
		AgentID:         "aara-ba-agent",
		ManifestVersion: "1.0.0",
		EngagementID:    "eng-example-001",
		UserID:          "user-example-001",
		TenantNamespace: "eng-example-001",
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "examples", "ba-agent.manifest.yaml")); err == nil {
			return wd
		}
		next := filepath.Dir(wd)
		if next == wd {
			t.Fatal("could not find repo root")
		}
		wd = next
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
