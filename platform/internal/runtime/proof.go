package aapruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func RunPhase1Proof(root string) (ProofReport, error) {
	engine, err := NewEngine(root, "examples/ba-agent.manifest.yaml", "tool-contracts")
	if err != nil {
		return ProofReport{}, err
	}
	ctx := RunContext{
		RunID:           "run-proof-001",
		AgentID:         "aara-ba-agent",
		ManifestVersion: "1.0.0",
		EngagementID:    "eng-example-001",
		UserID:          "user-example-001",
		TenantNamespace: "eng-example-001",
	}
	if err := engine.Start(ctx); err != nil {
		return ProofReport{}, err
	}

	allowed := engine.InvokeTool("get_project_context", map[string]any{
		"engagement_id": ctx.EngagementID,
		"query":         "sample requirements discovery acceptance criteria",
	}, RuntimeInteractive, true)

	denied := engine.InvokeTool("delete_production_record", map[string]any{
		"engagement_id": ctx.EngagementID,
		"record_id":     "sample-record",
	}, RuntimeInteractive, true)

	approval := engine.InvokeTool("create_requirements_draft", map[string]any{
		"engagement_id": ctx.EngagementID,
		"title":         "Sample discovery requirements",
		"evidence_refs": []string{"source://example/discovery-note"},
	}, RuntimeUnattended, false)

	err = engine.WriteMemory(MemoryRecord{
		MemoryID:       "mem-proof-001",
		AgentID:        ctx.AgentID,
		EngagementID:   ctx.EngagementID,
		Classification: "client-confidential",
		ContentRef:     "memory://eng-example-001/mem-proof-001",
		ContentHash:    hashPayload("sample cited memory"),
		SourceCitation: SourceCitation{
			RunID:     ctx.RunID,
			TraceID:   "trace-" + ctx.RunID,
			SpanID:    "span-proof-source",
			SourceRef: "source://example/discovery-note",
		},
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(90 * 24 * time.Hour),
		Status:    "active",
	})
	if err != nil {
		return ProofReport{}, err
	}

	leaked := engine.QueryMemory("eng-other-001")
	report := ProofReport{
		RunContext:               engine.RunContext(),
		ValidManifestStarted:     true,
		AllowedToolExecuted:      allowed.Outcome == "success",
		OffManifestToolDenied:    denied.Outcome == "denied",
		DenialAuditLogged:        hasAuditEvent(engine.AuditEvents(), "tool_denied", denied.AuditEventID),
		SoftUnattendedEscalated:  approval.Outcome == "approval_required" && approval.ApprovalBoundary == BoundaryHard,
		MemoryLeakageReturned:    len(leaked),
		RunScopedAuditsHaveRunID: AuditRunEventsHaveRunID(engine.AuditEvents()),
		TraceSpanCount:           len(engine.TraceSpans()),
		AuditEvents:              engine.AuditEvents(),
		ToolDecisions:            []ToolDecision{allowed, denied, approval},
	}
	return report, nil
}

func WriteProofReport(root string, report ProofReport, relPath string) error {
	outPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal proof report: %w", err)
	}
	return os.WriteFile(outPath, append(b, '\n'), 0o644)
}

func hasAuditEvent(events []AuditEvent, eventType, auditID string) bool {
	for _, event := range events {
		if event.EventType == eventType && event.AuditEventID == auditID {
			return true
		}
	}
	return false
}
