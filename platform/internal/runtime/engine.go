package aapruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	mu         sync.Mutex
	root       string
	manifest   Manifest
	contracts  map[string]ToolContract
	blocked    BlockedActions
	schemas    RuntimeSchemas
	run        RunContext
	audit      []AuditEvent
	traceSpans []TraceSpan
	memory     *MemoryStore
	nextID     int
}

func NewEngine(root, manifestPath, contractsDir string) (*Engine, error) {
	schemas, err := LoadRuntimeSchemas(root)
	if err != nil {
		return nil, err
	}
	manifestFile := filepath.Join(root, manifestPath)
	if err := ValidateStructuredFile(manifestFile, filepath.Join(root, "schemas", "agent-manifest.schema.json")); err != nil {
		return nil, err
	}
	manifest, err := LoadManifest(manifestFile)
	if err != nil {
		return nil, err
	}
	contracts, err := LoadContractsWithSchema(filepath.Join(root, contractsDir), filepath.Join(root, "schemas", "mcp-tool-contract.schema.json"))
	if err != nil {
		return nil, err
	}
	blocked, err := LoadBlockedActions(filepath.Join(root, manifest.ApprovalBoundaries.BlockedActionsRef))
	if err != nil {
		return nil, err
	}
	e := &Engine{
		root:      root,
		manifest:  manifest,
		contracts: contracts,
		blocked:   blocked,
		schemas:   schemas,
		memory:    NewMemoryStore(),
	}
	if err := e.validateManifest(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Start(ctx RunContext) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ctx.RunID == "" || ctx.EngagementID == "" || ctx.UserID == "" || ctx.TenantNamespace == "" {
		return errors.New("run_id, engagement_id, user_id, and tenant_namespace are required")
	}
	if ctx.AgentID == "" {
		ctx.AgentID = e.manifest.AgentID
	}
	if ctx.ManifestVersion == "" {
		ctx.ManifestVersion = e.manifest.ManifestVersion
	}
	if ctx.AgentID != e.manifest.AgentID {
		return fmt.Errorf("run agent_id %q does not match manifest agent_id %q", ctx.AgentID, e.manifest.AgentID)
	}
	if ctx.ManifestVersion != e.manifest.ManifestVersion {
		return fmt.Errorf("run manifest_version %q does not match manifest version %q", ctx.ManifestVersion, e.manifest.ManifestVersion)
	}
	e.run = ctx
	e.recordTrace("agent.run", "agent", map[string]any{
		"agent_id":         ctx.AgentID,
		"manifest_version": ctx.ManifestVersion,
		"engagement_id":    ctx.EngagementID,
	})
	if _, err := e.recordAudit("agent_started", "agent", ctx.AgentID, "agent", ctx.AgentID, map[string]any{
		"manifest_version": ctx.ManifestVersion,
	}); err != nil {
		return err
	}
	if _, err := e.recordAudit("manifest_validated", "run", ctx.RunID, "system", "aap-runtime", map[string]any{
		"manifest_version": ctx.ManifestVersion,
		"payload_mode":     e.manifest.Telemetry.PayloadMode,
	}); err != nil {
		return err
	}
	return nil
}

func (e *Engine) InvokeTool(toolName string, payload map[string]any, mode RuntimeMode, userConfirmed bool) ToolDecision {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.run.RunID == "" {
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "run has not started",
		}
	}
	allowed, manifestTool := e.allowedTool(toolName)
	if !allowed {
		event, err := e.recordAudit("tool_denied", "run", e.run.RunID, "agent", e.manifest.AgentID, map[string]any{
			"tool_name": toolName,
			"reason":    "off_manifest",
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		e.recordTrace("tool.denied", "tool", map[string]any{"tool_name": toolName, "reason": "off_manifest"})
		return ToolDecision{
			ToolName:     toolName,
			Outcome:      "denied",
			Reason:       "tool is not declared in manifest",
			AuditEventID: event.AuditEventID,
		}
	}

	contract, ok := e.contracts[toolName]
	if !ok {
		event, err := e.recordAudit("tool_denied", "run", e.run.RunID, "agent", e.manifest.AgentID, map[string]any{
			"tool_name": toolName,
			"reason":    "missing_contract",
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool contract is missing",
			AuditEventID:     event.AuditEventID,
		}
	}

	if contract.ContractVersion != manifestTool.ContractVersion {
		event, err := e.recordAudit("tool_denied", "run", e.run.RunID, "agent", e.manifest.AgentID, map[string]any{
			"tool_name": toolName,
			"reason":    "contract_version_mismatch",
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool contract version does not match manifest pin",
			AuditEventID:     event.AuditEventID,
		}
	}

	if e.isBlockedAction(contract.ActionType) {
		event, err := e.recordToolAudit("tool_denied", "agent", e.manifest.AgentID, contract, toolName, payload, "denied", map[string]any{
			"reason":      "blocked_action",
			"action_type": contract.ActionType,
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		e.recordTrace("tool.denied", "tool", map[string]any{"tool_name": toolName, "reason": "blocked_action", "action_type": contract.ActionType})
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool action type is blocked",
			AuditEventID:     event.AuditEventID,
		}
	}

	if err := validateValueAgainstSchema(payload, contract.InputSchema, toolName+" input"); err != nil {
		event, auditErr := e.recordToolAudit("tool_denied", "agent", e.manifest.AgentID, contract, toolName, payload, "denied", map[string]any{
			"reason": "input_schema_violation",
		})
		if auditErr != nil {
			return auditFailureDecision(toolName, auditErr)
		}
		e.recordTrace("tool.denied", "tool", map[string]any{"tool_name": toolName, "reason": "input_schema_violation"})
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool input does not match contract schema",
			AuditEventID:     event.AuditEventID,
		}
	}

	if payloadEngagement, ok := payload["engagement_id"].(string); ok && payloadEngagement != e.run.EngagementID {
		event, err := e.recordToolAudit("tool_denied", "agent", e.manifest.AgentID, contract, toolName, payload, "denied", map[string]any{
			"reason":            "engagement_scope_mismatch",
			"run_engagement_id": e.run.EngagementID,
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		e.recordTrace("tool.denied", "tool", map[string]any{"tool_name": toolName, "reason": "engagement_scope_mismatch"})
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool payload engagement_id does not match active run",
			AuditEventID:     event.AuditEventID,
		}
	}

	boundary := contract.ApprovalBoundary
	if manifestTool.ApprovalBoundary == BoundaryBlocked || boundary == BoundaryBlocked {
		event, err := e.recordToolAudit("tool_denied", "agent", e.manifest.AgentID, contract, toolName, payload, "denied", map[string]any{
			"reason":            "blocked_boundary",
			"approval_boundary": string(BoundaryBlocked),
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "tool is blocked in v1",
			AuditEventID:     event.AuditEventID,
		}
	}

	if boundary == BoundaryHard || (boundary == BoundarySoft && (mode == RuntimeUnattended || !userConfirmed)) {
		effective := boundary
		reason := "approval required"
		if boundary == BoundarySoft && mode == RuntimeUnattended {
			effective = BoundaryHard
			reason = "soft approval escalated to hard in unattended mode"
		}
		event, err := e.recordToolAudit("approval_requested", "agent", e.manifest.AgentID, contract, toolName, payload, "approval_required", map[string]any{
			"reason":            reason,
			"approval_boundary": string(effective),
			"original_boundary": string(boundary),
			"runtime_mode":      string(mode),
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		e.recordTrace("tool.approval_required", "tool", map[string]any{"tool_name": toolName, "boundary": effective})
		return ToolDecision{
			ToolName:         toolName,
			Outcome:          "approval_required",
			ApprovalBoundary: effective,
			Reason:           reason,
			AuditEventID:     event.AuditEventID,
		}
	}

	event, err := e.recordToolAudit("tool_invoked", "tool_principal", e.manifest.AgentID+":tool", contract, toolName, payload, "success", map[string]any{
		"approval_boundary": string(boundary),
	})
	if err != nil {
		return auditFailureDecision(toolName, err)
	}
	e.recordTrace("tool.invoked", "tool", map[string]any{"tool_name": toolName, "boundary": boundary})
	return ToolDecision{
		ToolName:         toolName,
		Outcome:          "success",
		ApprovalBoundary: boundary,
		Reason:           "tool executed by proof harness",
		AuditEventID:     event.AuditEventID,
	}
}

func (e *Engine) WriteMemory(record MemoryRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.run.RunID == "" {
		return errors.New("run has not started")
	}
	if !e.manifest.Memory.Enabled {
		return errors.New("memory is disabled by manifest")
	}
	if record.AgentID == "" {
		record.AgentID = e.manifest.AgentID
	}
	if record.AgentID != e.run.AgentID {
		return fmt.Errorf("memory agent %q does not match run agent %q", record.AgentID, e.run.AgentID)
	}
	if record.EngagementID != e.run.EngagementID {
		return fmt.Errorf("memory engagement %q does not match run engagement %q", record.EngagementID, e.run.EngagementID)
	}
	if record.SourceCitation.RunID != e.run.RunID {
		return fmt.Errorf("memory source run %q does not match active run %q", record.SourceCitation.RunID, e.run.RunID)
	}
	if record.Status == "" {
		record.Status = "active"
	}
	if err := e.schemas.ValidateMemoryRecord(record); err != nil {
		return err
	}
	if err := e.memory.Write(record, e.manifest.Memory.AllowedClassifications, e.manifest.Memory.PIIAllowed); err != nil {
		return err
	}
	if _, err := e.recordAudit("memory_written", "run", e.run.RunID, "agent", e.manifest.AgentID, map[string]any{
		"memory_id":      record.MemoryID,
		"engagement_id":  record.EngagementID,
		"classification": record.Classification,
		"content_hash":   record.ContentHash,
	}); err != nil {
		return err
	}
	return nil
}

func (e *Engine) QueryMemory(engagementID string) []MemoryRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.run.RunID == "" || engagementID != e.run.EngagementID {
		return nil
	}
	return e.memory.Query(engagementID, e.manifest.Memory.Scope, e.run.AgentID)
}

func (e *Engine) AuditEvents() []AuditEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]AuditEvent(nil), e.audit...)
}

func (e *Engine) TraceSpans() []TraceSpan {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]TraceSpan(nil), e.traceSpans...)
}

func (e *Engine) RunContext() RunContext {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.run
}

func (e *Engine) validateManifest() error {
	if e.manifest.AgentID == "" || e.manifest.ManifestVersion == "" || e.manifest.Owner == "" {
		return errors.New("manifest requires agent_id, manifest_version, and owner")
	}
	if e.manifest.Runtime != "claude-agent-sdk" {
		return fmt.Errorf("unsupported runtime %q", e.manifest.Runtime)
	}
	switch e.manifest.Status {
	case "draft", "active", "platform-ready", "deprecated", "blocked":
	default:
		return fmt.Errorf("unsupported manifest status %q", e.manifest.Status)
	}
	if (e.manifest.Status == "active" || e.manifest.Status == "platform-ready") && e.manifest.Telemetry.PayloadMode != "hash-and-reference" {
		return errors.New("active/platform-ready manifests must use payload_mode hash-and-reference")
	}
	if e.manifest.ApprovalBoundaries.Default != BoundaryHard && e.manifest.ApprovalBoundaries.Default != BoundaryBlocked {
		return errors.New("approval boundary default must be hard or blocked")
	}
	if e.manifest.Memory.Scope != "none" && e.manifest.Memory.Scope != "agent" && e.manifest.Memory.Scope != "engagement" {
		return fmt.Errorf("unsupported memory scope %q", e.manifest.Memory.Scope)
	}
	if e.blocked.MissingContractBoundary != BoundaryBlocked {
		return errors.New("missing contract boundary must be blocked")
	}
	if e.blocked.DefaultUnclassifiedBoundary != BoundaryHard && e.blocked.DefaultUnclassifiedBoundary != BoundaryBlocked {
		return errors.New("default unclassified boundary must be hard or blocked")
	}
	if e.blocked.Version == "" || len(e.blocked.BlockedActions) == 0 {
		return errors.New("blocked actions require version and at least one blocked action")
	}
	for _, skill := range e.manifest.AllowedSkills {
		if skill.SkillID == "" || skill.SkillVersion == "" || skill.SourcePath == "" {
			return errors.New("allowed skill requires skill_id, skill_version, and source_path")
		}
		if _, err := os.Stat(filepath.Join(e.root, skill.SourcePath)); err != nil {
			return fmt.Errorf("allowed skill source_path %q is not readable: %w", skill.SourcePath, err)
		}
	}
	for _, tool := range e.manifest.AllowedTools {
		if tool.ToolName == "" || tool.ContractVersion == "" {
			return errors.New("allowed tool requires tool_name and contract_version")
		}
		contract, ok := e.contracts[tool.ToolName]
		if !ok {
			return fmt.Errorf("allowed tool %q has no contract", tool.ToolName)
		}
		if contract.ContractVersion != tool.ContractVersion {
			return fmt.Errorf("allowed tool %q pins contract %q but loaded %q", tool.ToolName, tool.ContractVersion, contract.ContractVersion)
		}
		if contract.ApprovalBoundary != tool.ApprovalBoundary {
			return fmt.Errorf("allowed tool %q manifest boundary %q differs from contract boundary %q", tool.ToolName, tool.ApprovalBoundary, contract.ApprovalBoundary)
		}
	}
	if e.manifest.EvaluationGate.Required {
		if _, err := os.Stat(filepath.Join(e.root, e.manifest.EvaluationGate.BenchmarkRef)); err != nil {
			return fmt.Errorf("evaluation benchmark_ref %q is not readable: %w", e.manifest.EvaluationGate.BenchmarkRef, err)
		}
		if _, err := os.Stat(filepath.Join(e.root, e.manifest.EvaluationGate.ThresholdProfile)); err != nil {
			return fmt.Errorf("evaluation threshold_profile %q is not readable: %w", e.manifest.EvaluationGate.ThresholdProfile, err)
		}
	}
	return nil
}

func (e *Engine) allowedTool(toolName string) (bool, ManifestTool) {
	for _, tool := range e.manifest.AllowedTools {
		if tool.ToolName == toolName {
			return true, tool
		}
	}
	return false, ManifestTool{}
}

func (e *Engine) isBlockedAction(actionType string) bool {
	for _, blocked := range e.blocked.BlockedActions {
		if blocked == actionType {
			return true
		}
	}
	return false
}

func (e *Engine) recordToolAudit(eventType, actorType, actorID string, contract ToolContract, toolName string, input map[string]any, outcome string, extra map[string]any) (AuditEvent, error) {
	payload := map[string]any{
		"run_id":        e.run.RunID,
		"tool_name":     toolName,
		"engagement_id": e.run.EngagementID,
		"input_hash":    hashPayload(input),
		"outcome":       outcome,
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"action_type":   contract.ActionType,
	}
	if requestedEngagement, ok := input["engagement_id"].(string); ok && requestedEngagement != "" {
		payload["engagement_id"] = requestedEngagement
	}
	for key, value := range extra {
		payload[key] = value
	}
	if err := validateValueAgainstSchema(payload, contract.AuditEventSchema, toolName+" audit_event"); err != nil {
		return AuditEvent{}, err
	}
	return e.recordAudit(eventType, "run", e.run.RunID, actorType, actorID, payload)
}

func auditFailureDecision(toolName string, err error) ToolDecision {
	return ToolDecision{
		ToolName:         toolName,
		Outcome:          "denied",
		ApprovalBoundary: BoundaryBlocked,
		Reason:           "audit recording failed: " + err.Error(),
	}
}

func (e *Engine) recordAudit(eventType, contextType, contextID, actorType, actorID string, payload map[string]any) (AuditEvent, error) {
	e.nextID++
	if contextType == "run" && e.run.RunID == "" {
		return AuditEvent{}, errors.New("run-scoped audit cannot be recorded before run_id exists")
	}
	event := AuditEvent{
		AuditEventID: fmt.Sprintf("audit-%04d", e.nextID),
		EventType:    eventType,
		ActorType:    actorType,
		ActorID:      actorID,
		ContextType:  contextType,
		ContextID:    contextID,
		PayloadRef:   "hash://sha256/" + hashPayload(payload),
		PayloadHash:  hashPayload(payload),
		Timestamp:    time.Now().UTC(),
	}
	if contextType == "run" {
		event.RunID = e.run.RunID
	}
	if err := e.schemas.ValidateAuditEvent(event); err != nil {
		return AuditEvent{}, err
	}
	e.audit = append(e.audit, event)
	return event, nil
}

func (e *Engine) recordTrace(name, kind string, attrs map[string]any) {
	e.nextID++
	now := time.Now().UTC()
	e.traceSpans = append(e.traceSpans, TraceSpan{
		TraceID:   "trace-" + e.run.RunID,
		SpanID:    fmt.Sprintf("span-%04d", e.nextID),
		RunID:     e.run.RunID,
		Name:      name,
		Kind:      kind,
		StartTime: now,
		EndTime:   now,
		Attrs:     attrs,
	})
}

func hashPayload(payload any) string {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(fmt.Sprintf("%v", payload))
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func AuditRunEventsHaveRunID(events []AuditEvent) bool {
	for _, event := range events {
		if strings.EqualFold(event.ContextType, "run") && event.RunID == "" {
			return false
		}
	}
	return true
}
