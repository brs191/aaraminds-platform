package aapruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var tracerProviderTestMu sync.Mutex

func TestValidManifestStarts(t *testing.T) {
	engine := newTestEngine(t)
	if err := engine.Start(testRunContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(engine.AuditEvents()) == 0 {
		t.Fatal("expected audit events")
	}
}

func TestRepeatedStartFails(t *testing.T) {
	engine := newTestEngine(t)
	if err := engine.Start(testRunContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := engine.Start(testRunContext()); err == nil {
		t.Fatal("expected repeated start to fail")
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
agent_id: aara-business-analyst
manifest_version: 1.0.0
owner: Raja
runtime: claude-agent-sdk
status: draft
allowed_skills:
  - skill_id: aara-business-analyst-core
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
  "agent_id": "aara-business-analyst",
  "manifest_version": "1.0.0",
  "owner": "Raja",
  "runtime": "claude-agent-sdk",
  "status": "draft",
  "allowed_skills": [
    {
      "skill_id": "aara-business-analyst-core",
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

func TestToolResultAcceptedAfterSuccessfulInvocation(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	invocation := engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	if invocation.Outcome != "success" {
		t.Fatalf("expected successful invocation, got %+v", invocation)
	}
	decision := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if decision.Outcome != "success" {
		t.Fatalf("expected accepted result, got %+v", decision)
	}
	if !hasAuditEvent(engine.AuditEvents(), "tool_result_accepted", decision.AuditEventID) {
		t.Fatalf("expected result audit event, got %+v", engine.AuditEvents())
	}
}

func TestOpenTelemetrySpansIncludeGenAIAndAAPAttributes(t *testing.T) {
	tracerProviderTestMu.Lock()
	defer tracerProviderTestMu.Unlock()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	defer func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	invocation := engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	if invocation.Outcome != "success" {
		t.Fatalf("expected successful invocation, got %+v", invocation)
	}
	result := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if result.Outcome != "success" {
		t.Fatalf("expected accepted result, got %+v", result)
	}
	if err := engine.EndRun(context.Background()); err != nil {
		t.Fatalf("end run: %v", err)
	}

	spans := recorder.Ended()
	rootSpan := findEndedSpan(t, spans, "invoke_agent aara-business-analyst")
	toolSpan := findEndedSpan(t, spans, "tool.invoked")
	resultSpan := findEndedSpan(t, spans, "tool.result_accepted")
	if toolSpan.SpanKind() != trace.SpanKindClient {
		t.Fatalf("expected tool span kind client, got %s", toolSpan.SpanKind())
	}
	if rootSpan.SpanContext().TraceID() != toolSpan.SpanContext().TraceID() ||
		rootSpan.SpanContext().TraceID() != resultSpan.SpanContext().TraceID() {
		t.Fatalf("expected one run trace, got root=%s tool=%s result=%s",
			rootSpan.SpanContext().TraceID(),
			toolSpan.SpanContext().TraceID(),
			resultSpan.SpanContext().TraceID(),
		)
	}
	for _, span := range spans {
		if span.SpanContext().TraceID() != rootSpan.SpanContext().TraceID() {
			t.Fatalf("span %q escaped run trace: root=%s got=%s", span.Name(), rootSpan.SpanContext().TraceID(), span.SpanContext().TraceID())
		}
	}
	if toolSpan.Parent().SpanID() != rootSpan.SpanContext().SpanID() {
		t.Fatalf("expected tool span parent %s, got %s", rootSpan.SpanContext().SpanID(), toolSpan.Parent().SpanID())
	}
	if len(toolSpan.Events()) != 0 {
		t.Fatalf("expected no duplicated span events, got %+v", toolSpan.Events())
	}

	rootAttrs := spanAttributeStrings(rootSpan)
	if got := rootAttrs["gen_ai.operation.name"]; got != "invoke_agent" {
		t.Fatalf("expected root gen_ai.operation.name=invoke_agent, got %q in %+v", got, rootAttrs)
	}
	for _, moved := range []string{"gen_ai.agent.version", "gen_ai.workflow.name"} {
		if _, ok := rootAttrs[moved]; ok {
			t.Fatalf("span used moved/nonlocal GenAI attr %q in %+v", moved, rootAttrs)
		}
	}

	attrs := spanAttributeStrings(toolSpan)
	expected := map[string]string{
		"aap.run_id":            "run-test-001",
		"aap.engagement_id":     "eng-example-001",
		"aap.agent_id":          "aara-business-analyst",
		"aap.tool_name":         "get_project_context",
		"aap.audit_event_id":    invocation.AuditEventID,
		"gen_ai.agent.id":       "aara-business-analyst",
		"gen_ai.operation.name": "execute_tool",
		"gen_ai.tool.name":      "get_project_context",
		"gen_ai.tool.type":      "function",
	}
	for key, want := range expected {
		if got := attrs[key]; got != want {
			t.Fatalf("expected span attr %s=%q, got %q in %+v", key, want, got, attrs)
		}
	}
	resultAttrs := spanAttributeStrings(resultSpan)
	if got := resultAttrs["aap.audit_event_id"]; got != result.AuditEventID {
		t.Fatalf("expected result audit_event_id=%q, got %q in %+v", result.AuditEventID, got, resultAttrs)
	}
	if _, ok := resultAttrs["gen_ai.operation.name"]; ok {
		t.Fatalf("result span should not set gen_ai.operation.name: %+v", resultAttrs)
	}
	for _, disallowed := range []string{"aap.query", "aap.title", "aap.evidence_refs"} {
		if _, ok := attrs[disallowed]; ok {
			t.Fatalf("span leaked raw tool input attribute %q in %+v", disallowed, attrs)
		}
	}
}

func TestOTelHTTPSDefaultUsesSecureTransport(t *testing.T) {
	t.Setenv("AAP_OTEL_ENDPOINT", "https://collector.example:4317")
	t.Setenv("AAP_OTEL_INSECURE", "")
	cfg := OTelConfigFromEnv("test-service", "1.0.0")
	if cfg.Insecure {
		t.Fatalf("expected https endpoint to default to secure transport: %+v", cfg)
	}

	t.Setenv("AAP_OTEL_ENDPOINT", "localhost:4317")
	cfg = OTelConfigFromEnv("test-service", "1.0.0")
	if !cfg.Insecure {
		t.Fatalf("expected scheme-less local endpoint to default to insecure transport: %+v", cfg)
	}
}

func TestToolResultWithoutSuccessfulInvocationDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	decision := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied result without invocation, got %+v", decision)
	}
	if decision.Reason != "tool result has no successful invocation" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestToolResultOutputSchemaViolationDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	decision := engine.RecordToolResult("get_project_context", input, map[string]any{
		"results": []any{},
	}, time.Second, 1)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied output schema violation, got %+v", decision)
	}
	if decision.Reason != "tool output does not match contract schema" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestToolResultTimeoutViolationDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	decision := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), 6*time.Second, 1)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied timeout violation, got %+v", decision)
	}
	if decision.Reason != "tool execution exceeded contract timeout" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestToolResultRetryPolicyViolationDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	decision := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 4)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied retry policy violation, got %+v", decision)
	}
	if decision.Reason != "tool retry policy exceeded" {
		t.Fatalf("unexpected reason: %+v", decision)
	}
}

func TestDuplicateToolResultDenied(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	first := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if first.Outcome != "success" {
		t.Fatalf("expected first result accepted, got %+v", first)
	}
	duplicate := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if duplicate.Outcome != "denied" || duplicate.Reason != "tool result has no successful invocation" {
		t.Fatalf("expected duplicate result to be denied, got %+v", duplicate)
	}
}

func TestDeniedResultDoesNotConsumeInvocation(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	rejected := engine.RecordToolResult("get_project_context", input, map[string]any{"results": []any{}}, time.Second, 1)
	if rejected.Outcome != "denied" {
		t.Fatalf("expected schema-violating result to be denied, got %+v", rejected)
	}
	corrected := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if corrected.Outcome != "success" {
		t.Fatalf("expected corrected result to be accepted, got %+v", corrected)
	}
}

func TestEachInvocationAuthorizesOneResult(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	_ = engine.InvokeTool("get_project_context", input, RuntimeInteractive, true)
	first := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	second := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	third := engine.RecordToolResult("get_project_context", input, validProjectContextOutput(), time.Second, 1)
	if first.Outcome != "success" || second.Outcome != "success" {
		t.Fatalf("expected two invocations to authorize two results, got %+v / %+v", first, second)
	}
	if third.Outcome != "denied" {
		t.Fatalf("expected third result to be denied, got %+v", third)
	}
}

func TestUnclassifiedActionEscalatedToDefaultBoundary(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	contractsDir := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, manifestPath, `{
  "agent_id": "aara-business-analyst",
  "manifest_version": "1.0.0",
  "owner": "Raja",
  "runtime": "claude-agent-sdk",
  "status": "draft",
  "allowed_skills": [
    {
      "skill_id": "aara-business-analyst-core",
      "skill_version": "existing-package-baseline",
      "source_path": "skills-pack/agent-packages/aara-business-analyst/agent.md"
    }
  ],
  "allowed_tools": [
    {
      "tool_name": "novel_tool",
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
	writeFile(t, filepath.Join(contractsDir, "novel.contract.yaml"), `{
  "tool_name": "novel_tool",
  "contract_version": "1.0.0",
  "action_type": "novel_experimental_action",
  "purpose": "Action type not present in the classified actions registry.",
  "input_schema": {
    "type": "object",
    "required": ["engagement_id"],
    "properties": {
      "engagement_id": { "type": "string", "minLength": 1 }
    },
    "additionalProperties": false
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "status": { "type": "string" }
    },
    "additionalProperties": false
  },
  "permissions_required": ["novel:action"],
  "approval_boundary": "none",
  "data_classification": { "input": "internal", "output": "internal" },
  "failure_modes": [
    {
      "code": "UNAVAILABLE",
      "meaning": "Unavailable.",
      "retryable": true,
      "safe_user_message": "The tool is unavailable."
    }
  ],
  "timeout_ms": 5000,
  "timeout_class": "interactive",
  "retry_policy": { "max_attempts": 0, "backoff": "none" },
  "audit_event_schema": {
    "type": "object",
    "required": ["run_id", "tool_name", "engagement_id", "input_hash", "outcome", "timestamp"],
    "properties": {
      "run_id": { "type": "string", "minLength": 1 },
      "tool_name": { "type": "string", "const": "novel_tool" },
      "engagement_id": { "type": "string", "minLength": 1 },
      "input_hash": { "type": "string", "minLength": 1 },
      "outcome": { "type": "string", "enum": ["success", "denied", "approval_required"] },
      "timestamp": { "type": "string", "format": "date-time" },
      "reason": { "type": "string" },
      "action_type": { "type": "string" },
      "approval_boundary": { "type": "string" },
      "original_boundary": { "type": "string" },
      "runtime_mode": { "type": "string" },
      "approval_request_id": { "type": "string" },
      "approved_by": { "type": "string" }
    },
    "additionalProperties": false
  },
  "example_invocation": {
    "engagement_id": "eng-example-001"
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
	input := map[string]any{"engagement_id": "eng-example-001"}

	// Contract boundary is none, but the unclassified action type escalates
	// to default_unclassified_boundary (hard): no autonomous execution.
	decision := engine.InvokeTool("novel_tool", input, RuntimeInteractive, true)
	if decision.Outcome != "approval_required" || decision.ApprovalBoundary != BoundaryHard {
		t.Fatalf("expected unclassified action to require hard approval, got %+v", decision)
	}

	// A named grant still authorizes execution through the approval lifecycle.
	if err := engine.ResolveApproval(decision.ApprovalRequestID, "approver-001", true, "reviewed novel action"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	executed := engine.InvokeTool("novel_tool", input, RuntimeInteractive, true)
	if executed.Outcome != "success" {
		t.Fatalf("expected granted unclassified action to execute, got %+v", executed)
	}

	// With a blocked default, unclassified actions are denied outright.
	engine.blocked.DefaultUnclassifiedBoundary = BoundaryBlocked
	denied := engine.InvokeTool("novel_tool", input, RuntimeInteractive, true)
	if denied.Outcome != "denied" || denied.Reason != "tool action type is not classified and unclassified actions are blocked" {
		t.Fatalf("expected blocked unclassified action to be denied, got %+v", denied)
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
  "agent_id": "aara-business-analyst",
  "manifest_version": "1.0.0",
  "owner": "Raja",
  "runtime": "claude-agent-sdk",
  "status": "draft",
  "allowed_skills": [
    {
      "skill_id": "aara-business-analyst-core",
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
  "output_schema": {
    "type": "object",
    "properties": {
      "status": { "type": "string" }
    },
    "additionalProperties": false
  },
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

func TestApprovalLifecycleGrantExecutesOnce(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
		"evidence_refs": []string{"source://example/test"},
	}
	requested := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if requested.Outcome != "approval_required" || requested.ApprovalRequestID == "" {
		t.Fatalf("expected approval_required with request id, got %+v", requested)
	}
	if err := engine.ResolveApproval(requested.ApprovalRequestID, "approver-001", true, "reviewed"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	executed := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if executed.Outcome != "success" {
		t.Fatalf("expected granted invocation to execute, got %+v", executed)
	}
	if executed.ApprovalRequestID != requested.ApprovalRequestID {
		t.Fatalf("expected execution to reference grant %q, got %+v", requested.ApprovalRequestID, executed)
	}
	if !hasAuditEventType(engine.AuditEvents(), "approval_granted", "approver") {
		t.Fatalf("expected approval_granted audit event with approver actor, got %+v", engine.AuditEvents())
	}
	repeat := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if repeat.Outcome != "approval_required" || repeat.ApprovalRequestID == requested.ApprovalRequestID {
		t.Fatalf("expected consumed grant to require fresh approval, got %+v", repeat)
	}
}

func TestApprovalDeniedDoesNotExecute(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
		"evidence_refs": []string{"source://example/test"},
	}
	requested := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if err := engine.ResolveApproval(requested.ApprovalRequestID, "approver-001", false, "insufficient evidence"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	if !hasAuditEventType(engine.AuditEvents(), "approval_denied", "approver") {
		t.Fatalf("expected approval_denied audit event, got %+v", engine.AuditEvents())
	}
	retry := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if retry.Outcome != "approval_required" {
		t.Fatalf("expected denied approval to not authorize execution, got %+v", retry)
	}
}

func TestResolveApprovalRequiresApproverAndPendingRequest(t *testing.T) {
	engine := newStartedTestEngine(t)
	input := map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
		"evidence_refs": []string{"source://example/test"},
	}
	requested := engine.InvokeTool("create_requirements_draft", input, RuntimeUnattended, false)
	if err := engine.ResolveApproval(requested.ApprovalRequestID, "", true, "no approver"); err == nil {
		t.Fatal("expected empty approver_id to be rejected")
	}
	if err := engine.ResolveApproval("approval-missing", "approver-001", true, "unknown"); err == nil {
		t.Fatal("expected unknown approval request to be rejected")
	}
	if err := engine.ResolveApproval(requested.ApprovalRequestID, "approver-001", true, "reviewed"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}
	if err := engine.ResolveApproval(requested.ApprovalRequestID, "approver-002", true, "double"); err == nil {
		t.Fatal("expected already-resolved request to be rejected")
	}
}

func TestSoftInteractiveConfirmationAuditsGrant(t *testing.T) {
	engine := newStartedTestEngine(t)
	decision := engine.InvokeTool("create_requirements_draft", map[string]any{
		"engagement_id": "eng-example-001",
		"title":         "Draft",
		"evidence_refs": []string{"source://example/test"},
	}, RuntimeInteractive, true)
	if decision.Outcome != "success" {
		t.Fatalf("expected confirmed soft invocation to execute, got %+v", decision)
	}
	if decision.ApprovalRequestID == "" {
		t.Fatalf("expected execution to reference an approval request, got %+v", decision)
	}
	found := false
	for _, event := range engine.AuditEvents() {
		if event.EventType == "approval_granted" && event.ActorType == "approver" && event.ActorID == testRunContext().UserID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected approval_granted attributed to run user, got %+v", engine.AuditEvents())
	}
}

func TestAuditTrailReplayableAndChainValid(t *testing.T) {
	engine := newStartedTestEngine(t)
	_ = engine.InvokeTool("get_project_context", map[string]any{
		"engagement_id": "eng-example-001",
		"query":         "sample requirements discovery acceptance criteria",
	}, RuntimeInteractive, true)
	_ = engine.InvokeTool("unknown_tool", map[string]any{"engagement_id": "eng-example-001"}, RuntimeInteractive, true)
	events := engine.AuditEvents()
	payloads := engine.AuditPayloads()
	if !VerifyAuditTrail(events, payloads) {
		t.Fatal("expected audit trail to be replayable")
	}
	if !VerifyAuditChain(events) {
		t.Fatal("expected audit chain to be valid")
	}
	// Tampering with any recorded event must break the chain.
	tampered := append([]AuditEvent(nil), events...)
	tampered[0].ActorID = "attacker"
	if VerifyAuditChain(tampered) {
		t.Fatal("expected tampered audit chain to fail verification")
	}
	// Deleting an event must break the chain.
	truncated := append([]AuditEvent(nil), events[:1]...)
	truncated = append(truncated, events[2:]...)
	if VerifyAuditChain(truncated) {
		t.Fatal("expected audit chain with deleted event to fail verification")
	}
	// A payload missing from the store must fail replay verification.
	delete(payloads, events[0].PayloadHash)
	if VerifyAuditTrail(events, payloads) {
		t.Fatal("expected missing payload to fail replay verification")
	}
}

func TestContractWithoutEngagementScopeRejectedAtLoad(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	contractsDir := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(contractsDir, "unscoped.contract.yaml"), `{
  "tool_name": "unscoped_tool",
  "contract_version": "1.0.0",
  "action_type": "knowledge_read",
  "purpose": "Contract missing the mandatory engagement_id input requirement.",
  "input_schema": {
    "type": "object",
    "required": ["query"],
    "properties": {
      "query": { "type": "string", "minLength": 1 }
    },
    "additionalProperties": false
  },
  "output_schema": {
    "type": "object",
    "required": ["results"],
    "properties": {
      "results": { "type": "array" }
    },
    "additionalProperties": false
  },
  "permissions_required": ["knowledge:read"],
  "approval_boundary": "none",
  "data_classification": { "input": "internal", "output": "internal" },
  "failure_modes": [
    {
      "code": "UNAVAILABLE",
      "meaning": "Source unavailable.",
      "retryable": true,
      "safe_user_message": "The source is unavailable."
    }
  ],
  "timeout_ms": 5000,
  "timeout_class": "interactive",
  "retry_policy": { "max_attempts": 1, "backoff": "fixed" },
  "audit_event_schema": {
    "type": "object",
    "required": ["run_id", "tool_name", "engagement_id", "input_hash", "outcome", "timestamp"],
    "properties": {
      "run_id": { "type": "string", "minLength": 1 },
      "tool_name": { "type": "string", "const": "unscoped_tool" },
      "engagement_id": { "type": "string", "minLength": 1 },
      "input_hash": { "type": "string", "minLength": 1 },
      "outcome": { "type": "string", "enum": ["success", "denied", "approval_required"] },
      "timestamp": { "type": "string", "format": "date-time" },
      "reason": { "type": "string" }
    },
    "additionalProperties": false
  },
  "example_invocation": {
    "query": "sample query"
  }
}`)
	contractsRel, err := filepath.Rel(root, contractsDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadContractsWithSchema(filepath.Join(root, contractsRel), filepath.Join(root, "schemas", "mcp-tool-contract.schema.json"))
	if err == nil {
		t.Fatal("expected contract without engagement_id requirement to be rejected at load")
	}
	if !strings.Contains(err.Error(), "engagement_id") {
		t.Fatalf("expected engagement_id load error, got: %v", err)
	}
}

func TestToolPayloadMissingEngagementDeniedFailClosed(t *testing.T) {
	engine := newStartedTestEngine(t)
	// Bypass the loader invariant to simulate an injected contract whose
	// input_schema does not require engagement_id; the runtime gate must
	// still fail closed instead of skipping the scope check.
	contract := engine.contracts["get_project_context"]
	contract.InputSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"additionalProperties": true,
	}
	engine.contracts["get_project_context"] = contract
	decision := engine.InvokeTool("get_project_context", map[string]any{
		"query": "sample requirements discovery acceptance criteria",
	}, RuntimeInteractive, true)
	if decision.Outcome != "denied" {
		t.Fatalf("expected denied for missing engagement_id, got %+v", decision)
	}
	if decision.Reason != "tool payload engagement_id does not match active run" {
		t.Fatalf("unexpected reason: %+v", decision)
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

func TestExpiredMemoryIsHidden(t *testing.T) {
	store := NewMemoryStore()
	record := MemoryRecord{
		MemoryID:       "mem-expired-001",
		AgentID:        "aara-business-analyst",
		EngagementID:   "eng-example-001",
		Classification: "client-confidential",
		ContentRef:     "memory://eng-example-001/mem-expired-001",
		ContentHash:    "hash-test",
		SourceCitation: SourceCitation{
			RunID:     "run-test-001",
			TraceID:   "trace-run-test-001",
			SpanID:    "span-test",
			SourceRef: "source://example/test",
		},
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
		Status:    "active",
	}
	store.records = append(store.records, record)
	if got := len(store.Query(record.EngagementID, "engagement", record.AgentID)); got != 0 {
		t.Fatalf("expected expired memory to be hidden, got %d", got)
	}
}

func TestExpiredMemoryWriteRejected(t *testing.T) {
	engine := newStartedTestEngine(t)
	ctx := testRunContext()
	err := engine.WriteMemory(MemoryRecord{
		MemoryID:       "mem-expired-002",
		AgentID:        ctx.AgentID,
		EngagementID:   ctx.EngagementID,
		Classification: "client-confidential",
		ContentRef:     "memory://eng-example-001/mem-expired-002",
		ContentHash:    "hash-test",
		SourceCitation: SourceCitation{
			RunID:     ctx.RunID,
			TraceID:   "trace-" + ctx.RunID,
			SpanID:    "span-test",
			SourceRef: "source://example/test",
		},
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
		Status:    "active",
	})
	if err == nil {
		t.Fatal("expected expired memory write to fail")
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
		!report.ToolOutputAccepted ||
		!report.OffManifestToolDenied ||
		!report.BlockedActionDenied ||
		!report.InvalidInputDenied ||
		!report.OutputSchemaViolationDenied ||
		!report.TimeoutViolationDenied ||
		!report.RetryPolicyViolationDenied ||
		!report.DuplicateResultDenied ||
		!report.DenialAuditLogged ||
		!report.SoftUnattendedEscalated ||
		!report.ApprovalGrantAudited ||
		!report.ApprovedInvocationExecuted ||
		!report.ApprovalGrantSingleUse ||
		!report.AuditTrailReplayable ||
		!report.AuditChainValid ||
		report.MemoryLeakageReturned != 0 ||
		report.ExpiredMemoryReturned != 0 ||
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
		AgentID:         "aara-business-analyst",
		ManifestVersion: "1.0.0",
		EngagementID:    "eng-example-001",
		UserID:          "user-example-001",
		TenantNamespace: "eng-example-001",
	}
}

func validProjectContextOutput() map[string]any {
	return map[string]any{
		"results": []any{
			map[string]any{
				"title":      "Discovery note",
				"excerpt":    "Stakeholders need approval-gated requirements drafts.",
				"source_ref": "source://example/discovery-note",
				"confidence": 0.94,
			},
		},
		"source_system": "proof-fixture",
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

func findEndedSpan(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("span %q not found in %d spans", name, len(spans))
	return nil
}

func spanAttributeStrings(span sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string)
	for _, attr := range span.Attributes() {
		out[string(attr.Key)] = attr.Value.AsString()
	}
	return out
}
