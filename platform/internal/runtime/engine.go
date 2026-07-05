package aapruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	mu            sync.Mutex
	root          string
	manifest      Manifest
	contracts     map[string]ToolContract
	blocked       BlockedActions
	schemas       RuntimeSchemas
	run           RunContext
	audit         []AuditEvent
	payloads      map[string]json.RawMessage
	lastEventHash string
	approvals     map[string]*ApprovalRequest
	traceSpans    []TraceSpan
	otelRun       otelRunState
	memory        *MemoryStore
	invoked       map[string]int
	nextID        int
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
		invoked:   make(map[string]int),
		payloads:  make(map[string]json.RawMessage),
		approvals: make(map[string]*ApprovalRequest),
	}
	if err := e.validateManifest(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Start(ctx RunContext) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.run.RunID != "" {
		return fmt.Errorf("run %q has already started", e.run.RunID)
	}
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
	e.startOTelRun(time.Now().UTC())
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

	contract, boundary, decision, ok := e.gateTool(toolName, payload)
	if !ok {
		return decision
	}

	invocationKey := toolInvocationKey(toolName, payload)

	// A previously granted approval for this exact tool + payload executes
	// and consumes the grant, regardless of runtime mode. Grants are
	// single-use: a repeat invocation goes back through the approval gate.
	if approvalID, approverID, granted := e.consumeGrant(invocationKey); granted {
		return e.executeInvocation(contract, toolName, payload, boundary, approvalID, approverID)
	}

	if boundary == BoundaryHard || (boundary == BoundarySoft && (mode == RuntimeUnattended || !userConfirmed)) {
		effective := boundary
		reason := "approval required"
		if boundary == BoundarySoft && mode == RuntimeUnattended {
			effective = BoundaryHard
			reason = "soft approval escalated to hard in unattended mode"
		}
		request, err := e.createApprovalRequest(contract, toolName, invocationKey, effective, mode)
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		event, err := e.recordToolAudit("approval_requested", "agent", e.manifest.AgentID, contract, toolName, payload, "approval_required", map[string]any{
			"reason":              reason,
			"approval_boundary":   string(effective),
			"original_boundary":   string(boundary),
			"runtime_mode":        string(mode),
			"approval_request_id": request.ApprovalRequestID,
		})
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		e.recordTrace("tool.approval_required", "tool", map[string]any{
			"tool_name":           toolName,
			"boundary":            effective,
			"approval_request_id": request.ApprovalRequestID,
			"audit_event_id":      event.AuditEventID,
		})
		return ToolDecision{
			ToolName:          toolName,
			Outcome:           "approval_required",
			ApprovalBoundary:  effective,
			Reason:            reason,
			AuditEventID:      event.AuditEventID,
			ApprovalRequestID: request.ApprovalRequestID,
		}
	}

	if boundary == BoundarySoft {
		// Interactive confirmation of a soft boundary is itself an approval
		// decision: record who granted it (the run user) before executing,
		// so no approval-gated execution lacks an approval_granted event.
		request, err := e.createApprovalRequest(contract, toolName, invocationKey, boundary, mode)
		if err != nil {
			return auditFailureDecision(toolName, err)
		}
		if err := e.resolveApprovalLocked(request.ApprovalRequestID, e.run.UserID, true, "confirmed interactively by run user"); err != nil {
			return auditFailureDecision(toolName, err)
		}
		approvalID, approverID, _ := e.consumeGrant(invocationKey)
		return e.executeInvocation(contract, toolName, payload, boundary, approvalID, approverID)
	}

	return e.executeInvocation(contract, toolName, payload, boundary, "", "")
}

func (e *Engine) executeInvocation(contract ToolContract, toolName string, payload map[string]any, boundary Boundary, approvalRequestID, approvedBy string) ToolDecision {
	extra := map[string]any{"approval_boundary": string(boundary)}
	traceAttrs := map[string]any{"tool_name": toolName, "boundary": boundary}
	if approvalRequestID != "" {
		extra["approval_request_id"] = approvalRequestID
		extra["approved_by"] = approvedBy
		traceAttrs["approval_request_id"] = approvalRequestID
	}
	event, err := e.recordToolAudit("tool_invoked", "tool_principal", e.manifest.AgentID+":tool", contract, toolName, payload, "success", extra)
	if err != nil {
		return auditFailureDecision(toolName, err)
	}
	e.invoked[toolInvocationKey(toolName, payload)]++
	traceAttrs["audit_event_id"] = event.AuditEventID
	e.recordTrace("tool.invoked", "tool", traceAttrs)
	return ToolDecision{
		ToolName:          toolName,
		Outcome:           "success",
		ApprovalBoundary:  boundary,
		Reason:            "tool executed by proof harness",
		AuditEventID:      event.AuditEventID,
		ApprovalRequestID: approvalRequestID,
	}
}

func (e *Engine) createApprovalRequest(contract ToolContract, toolName, invocationKey string, boundary Boundary, mode RuntimeMode) (*ApprovalRequest, error) {
	e.nextID++
	request := &ApprovalRequest{
		ApprovalRequestID: fmt.Sprintf("approval-%04d", e.nextID),
		RunID:             e.run.RunID,
		ToolInvocationID:  invocationKey,
		ApprovalBoundary:  boundary,
		RequestedAction:   toolName,
		RiskSummary:       contract.Purpose,
		RuntimeMode:       mode,
		Status:            "pending",
		CreatedAt:         time.Now().UTC(),
	}
	if err := e.schemas.ValidateApprovalRequest(*request); err != nil {
		return nil, err
	}
	e.approvals[request.ApprovalRequestID] = request
	return request, nil
}

// ResolveApproval records a human approval decision for a pending request,
// with the approver's identity, and emits an approval_granted or
// approval_denied audit event. A granted request authorizes exactly one
// subsequent invocation of the same tool with the same payload.
func (e *Engine) ResolveApproval(approvalRequestID, approverID string, approve bool, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.resolveApprovalLocked(approvalRequestID, approverID, approve, reason)
}

func (e *Engine) resolveApprovalLocked(approvalRequestID, approverID string, approve bool, reason string) error {
	if e.run.RunID == "" {
		return errors.New("run has not started")
	}
	if approverID == "" {
		return errors.New("approver_id is required")
	}
	request, ok := e.approvals[approvalRequestID]
	if !ok {
		return fmt.Errorf("approval request %q does not exist", approvalRequestID)
	}
	if request.Status != "pending" {
		return fmt.Errorf("approval request %q is already %s", approvalRequestID, request.Status)
	}
	now := time.Now().UTC()
	updated := *request
	updated.ApproverID = approverID
	updated.DecisionReason = reason
	updated.DecidedAt = &now
	if approve {
		updated.Status = "approved"
	} else {
		updated.Status = "denied"
	}
	if err := e.schemas.ValidateApprovalRequest(updated); err != nil {
		return err
	}
	eventType := "approval_granted"
	if !approve {
		eventType = "approval_denied"
	}
	if _, err := e.recordAudit(eventType, "run", e.run.RunID, "approver", approverID, map[string]any{
		"approval_request_id": updated.ApprovalRequestID,
		"tool_invocation_id":  updated.ToolInvocationID,
		"requested_action":    updated.RequestedAction,
		"approval_boundary":   string(updated.ApprovalBoundary),
		"decision_reason":     reason,
	}); err != nil {
		return err
	}
	*request = updated
	e.recordTrace("approval.resolved", "approval", map[string]any{
		"approval_request_id": updated.ApprovalRequestID,
		"status":              updated.Status,
		"audit_event_type":    eventType,
	})
	return nil
}

func (e *Engine) consumeGrant(invocationKey string) (string, string, bool) {
	for id, request := range e.approvals {
		if request.ToolInvocationID == invocationKey && request.Status == "approved" && !request.consumed {
			request.consumed = true
			return id, request.ApproverID, true
		}
	}
	return "", "", false
}

// ApprovalRequests returns a copy of all approval requests, ordered by ID.
func (e *Engine) ApprovalRequests() []ApprovalRequest {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make([]ApprovalRequest, 0, len(e.approvals))
	for _, request := range e.approvals {
		out = append(out, *request)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ApprovalRequestID < out[j].ApprovalRequestID })
	return out
}

func (e *Engine) RecordToolResult(toolName string, input, output map[string]any, elapsed time.Duration, attempts int) ToolDecision {
	e.mu.Lock()
	defer e.mu.Unlock()

	contract, boundary, decision, ok := e.gateTool(toolName, input)
	if !ok {
		return decision
	}
	if e.invoked[toolInvocationKey(toolName, input)] < 1 {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "missing_successful_invocation", "tool result has no successful invocation")
	}
	if attempts < 1 {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "attempts must be at least 1", "tool result attempts must be at least 1")
	}
	if err := validateValueAgainstSchema(output, contract.OutputSchema, toolName+" output"); err != nil {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "output_schema_violation", "tool output does not match contract schema")
	}
	if elapsed < 0 {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "negative_elapsed", "tool elapsed duration must not be negative")
	}
	if elapsed > time.Duration(contract.TimeoutMS)*time.Millisecond {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "timeout_exceeded", "tool execution exceeded contract timeout")
	}
	allowedAttempts := 1 + contract.RetryPolicy.MaxAttempts
	if attempts > allowedAttempts {
		return e.resultDenied(contract, toolName, input, output, elapsed, attempts, "retry_policy_exceeded", "tool retry policy exceeded")
	}

	event, err := e.recordToolAudit("tool_result_accepted", "tool_principal", e.manifest.AgentID+":tool", contract, toolName, input, "success", map[string]any{
		"approval_boundary": string(boundary),
		"output_hash":       hashPayload(output),
		"elapsed_ms":        elapsed.Milliseconds(),
		"attempts":          attempts,
	})
	if err != nil {
		return auditFailureDecision(toolName, err)
	}
	// An accepted result consumes its invocation: each successful invocation
	// authorizes exactly one accepted result. Denied results do not consume,
	// so a corrected result can still be recorded for the same invocation.
	e.invoked[toolInvocationKey(toolName, input)]--
	e.recordTrace("tool.result_accepted", "tool", map[string]any{
		"tool_name":      toolName,
		"boundary":       boundary,
		"elapsed_ms":     elapsed.Milliseconds(),
		"attempts":       attempts,
		"audit_event_id": event.AuditEventID,
	})
	return ToolDecision{
		ToolName:         toolName,
		Outcome:          "success",
		ApprovalBoundary: boundary,
		Reason:           "tool output accepted by proof harness",
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
	e.recordTrace("memory.written", "memory", map[string]any{
		"memory_id":      record.MemoryID,
		"engagement_id":  record.EngagementID,
		"classification": record.Classification,
	})
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
	for _, classified := range e.blocked.ClassifiedActions {
		if e.isBlockedAction(classified) {
			return fmt.Errorf("action type %q cannot be both blocked and classified", classified)
		}
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

func (e *Engine) isClassifiedAction(actionType string) bool {
	for _, classified := range e.blocked.ClassifiedActions {
		if classified == actionType {
			return true
		}
	}
	return false
}

// gateTool runs the shared admission checks for both tool invocation and tool
// result recording: run state, manifest allowlist, contract presence, version
// pin, blocked actions, input schema, engagement scope, and blocked
// boundaries. On success it returns the contract and its approval boundary;
// on any failure it records a tool_denied audit event plus a trace span and
// returns a denial decision with ok=false. Keeping this in one place
// guarantees the invoke and result paths can never drift apart.
func (e *Engine) gateTool(toolName string, payload map[string]any) (ToolContract, Boundary, ToolDecision, bool) {
	if e.run.RunID == "" {
		return ToolContract{}, BoundaryBlocked, ToolDecision{
			ToolName:         toolName,
			Outcome:          "denied",
			ApprovalBoundary: BoundaryBlocked,
			Reason:           "run has not started",
		}, false
	}
	allowed, manifestTool := e.allowedTool(toolName)
	if !allowed {
		return e.gateDenied(toolName, "off_manifest", "tool is not declared in manifest", nil)
	}
	contract, ok := e.contracts[toolName]
	if !ok {
		return e.gateDenied(toolName, "missing_contract", "tool contract is missing", nil)
	}
	if contract.ContractVersion != manifestTool.ContractVersion {
		return e.gateDenied(toolName, "contract_version_mismatch", "tool contract version does not match manifest pin", nil)
	}
	if e.isBlockedAction(contract.ActionType) {
		return e.gateDeniedWithContract(contract, toolName, payload, "blocked_action", "tool action type is blocked", map[string]any{
			"action_type": contract.ActionType,
		})
	}
	if err := validateValueAgainstSchema(payload, contract.InputSchema, toolName+" input"); err != nil {
		return e.gateDeniedWithContract(contract, toolName, payload, "input_schema_violation", "tool input does not match contract schema", nil)
	}
	payloadEngagement, isString := payload["engagement_id"].(string)
	if !isString || payloadEngagement != e.run.EngagementID {
		return e.gateDeniedWithContract(contract, toolName, payload, "engagement_scope_mismatch", "tool payload engagement_id does not match active run", map[string]any{
			"run_engagement_id": e.run.EngagementID,
		})
	}
	boundary := contract.ApprovalBoundary
	if manifestTool.ApprovalBoundary == BoundaryBlocked || boundary == BoundaryBlocked {
		return e.gateDeniedWithContract(contract, toolName, payload, "blocked_boundary", "tool is blocked in v1", map[string]any{
			"approval_boundary": string(BoundaryBlocked),
		})
	}
	// An action type that is neither blocked nor classified falls under
	// default_unclassified_boundary: blocked denies outright, hard escalates
	// the effective boundary so the tool cannot execute autonomously with a
	// weaker contract boundary.
	if !e.isClassifiedAction(contract.ActionType) {
		if e.blocked.DefaultUnclassifiedBoundary == BoundaryBlocked {
			return e.gateDeniedWithContract(contract, toolName, payload, "unclassified_action", "tool action type is not classified and unclassified actions are blocked", map[string]any{
				"action_type": contract.ActionType,
			})
		}
		if boundaryRank(e.blocked.DefaultUnclassifiedBoundary) > boundaryRank(boundary) {
			e.recordTrace("tool.boundary_escalated", "tool", map[string]any{
				"tool_name":         toolName,
				"reason":            "unclassified_action",
				"action_type":       contract.ActionType,
				"contract_boundary": boundary,
				"applied_boundary":  e.blocked.DefaultUnclassifiedBoundary,
			})
			boundary = e.blocked.DefaultUnclassifiedBoundary
		}
	}
	return contract, boundary, ToolDecision{}, true
}

func boundaryRank(boundary Boundary) int {
	switch boundary {
	case BoundaryNone:
		return 0
	case BoundarySoft:
		return 1
	case BoundaryHard:
		return 2
	case BoundaryBlocked:
		return 3
	default:
		return 3
	}
}

// gateDenied records a denial that happens before a valid contract is
// resolved, so the audit event cannot be validated against a contract-specific
// audit_event_schema and uses the generic run-scoped audit path instead.
func (e *Engine) gateDenied(toolName, reason, userReason string, extra map[string]any) (ToolContract, Boundary, ToolDecision, bool) {
	auditPayload := map[string]any{
		"tool_name": toolName,
		"reason":    reason,
	}
	for key, value := range extra {
		auditPayload[key] = value
	}
	event, err := e.recordAudit("tool_denied", "run", e.run.RunID, "agent", e.manifest.AgentID, auditPayload)
	if err != nil {
		return ToolContract{}, BoundaryBlocked, auditFailureDecision(toolName, err), false
	}
	e.recordTrace("tool.denied", "tool", map[string]any{"tool_name": toolName, "reason": reason, "audit_event_id": event.AuditEventID})
	return ToolContract{}, BoundaryBlocked, ToolDecision{
		ToolName:         toolName,
		Outcome:          "denied",
		ApprovalBoundary: BoundaryBlocked,
		Reason:           userReason,
		AuditEventID:     event.AuditEventID,
	}, false
}

// gateDeniedWithContract records a denial for a resolved contract, validating
// the audit payload against the contract's audit_event_schema.
func (e *Engine) gateDeniedWithContract(contract ToolContract, toolName string, payload map[string]any, reason, userReason string, extra map[string]any) (ToolContract, Boundary, ToolDecision, bool) {
	auditExtra := map[string]any{"reason": reason}
	traceAttrs := map[string]any{"tool_name": toolName, "reason": reason}
	for key, value := range extra {
		auditExtra[key] = value
		traceAttrs[key] = value
	}
	event, err := e.recordToolAudit("tool_denied", "agent", e.manifest.AgentID, contract, toolName, payload, "denied", auditExtra)
	if err != nil {
		return ToolContract{}, BoundaryBlocked, auditFailureDecision(toolName, err), false
	}
	traceAttrs["audit_event_id"] = event.AuditEventID
	e.recordTrace("tool.denied", "tool", traceAttrs)
	return ToolContract{}, BoundaryBlocked, ToolDecision{
		ToolName:         toolName,
		Outcome:          "denied",
		ApprovalBoundary: BoundaryBlocked,
		Reason:           userReason,
		AuditEventID:     event.AuditEventID,
	}, false
}

func (e *Engine) resultDenied(contract ToolContract, toolName string, input, output map[string]any, elapsed time.Duration, attempts int, reason, userReason string) ToolDecision {
	event, err := e.recordToolAudit("tool_result_denied", "agent", e.manifest.AgentID, contract, toolName, input, "denied", map[string]any{
		"reason":      reason,
		"output_hash": hashPayload(output),
		"elapsed_ms":  elapsed.Milliseconds(),
		"attempts":    attempts,
	})
	if err != nil {
		return auditFailureDecision(toolName, err)
	}
	e.recordTrace("tool.result_denied", "tool", map[string]any{
		"tool_name":      toolName,
		"reason":         reason,
		"elapsed_ms":     elapsed.Milliseconds(),
		"attempts":       attempts,
		"audit_event_id": event.AuditEventID,
	})
	return ToolDecision{
		ToolName:         toolName,
		Outcome:          "denied",
		ApprovalBoundary: BoundaryBlocked,
		Reason:           userReason,
		AuditEventID:     event.AuditEventID,
	}
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
	canonical, hash := canonicalPayload(payload)
	event := AuditEvent{
		AuditEventID:  fmt.Sprintf("audit-%04d", e.nextID),
		EventType:     eventType,
		ActorType:     actorType,
		ActorID:       actorID,
		ContextType:   contextType,
		ContextID:     contextID,
		PayloadRef:    "cas://sha256/" + hash,
		PayloadHash:   hash,
		PrevEventHash: e.lastEventHash,
		Timestamp:     time.Now().UTC(),
	}
	if contextType == "run" {
		event.RunID = e.run.RunID
	}
	if err := e.schemas.ValidateAuditEvent(event); err != nil {
		return AuditEvent{}, err
	}
	// Persist the payload in the content-addressed store so payload_ref
	// resolves and the run is replayable, then extend the tamper-evident
	// hash chain.
	e.payloads[hash] = json.RawMessage(canonical)
	e.audit = append(e.audit, event)
	e.lastEventHash = hashEvent(event)
	return event, nil
}

// AuditPayloads returns a copy of the content-addressed payload store keyed
// by sha256 hash. Together with AuditEvents this makes the audit trail
// replayable: every payload_ref resolves to stored canonical JSON.
func (e *Engine) AuditPayloads() map[string]json.RawMessage {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make(map[string]json.RawMessage, len(e.payloads))
	for hash, payload := range e.payloads {
		out[hash] = append(json.RawMessage(nil), payload...)
	}
	return out
}

// VerifyAuditTrail checks that every audit event's payload_hash resolves in
// the payload store and that the stored payload still hashes to it.
func VerifyAuditTrail(events []AuditEvent, payloads map[string]json.RawMessage) bool {
	for _, event := range events {
		payload, ok := payloads[event.PayloadHash]
		if !ok {
			return false
		}
		sum := sha256.Sum256(payload)
		if hex.EncodeToString(sum[:]) != event.PayloadHash {
			return false
		}
		if event.PayloadRef != "cas://sha256/"+event.PayloadHash {
			return false
		}
	}
	return true
}

// VerifyAuditChain checks the tamper-evident hash chain: each event's
// prev_event_hash must equal the hash of the preceding event, starting from
// an empty genesis hash. Any insertion, deletion, reordering, or mutation of
// an earlier event breaks the chain.
func VerifyAuditChain(events []AuditEvent) bool {
	prev := ""
	for _, event := range events {
		if event.PrevEventHash != prev {
			return false
		}
		prev = hashEvent(event)
	}
	return true
}

func hashEvent(event AuditEvent) string {
	b, err := json.Marshal(event)
	if err != nil {
		b = []byte(fmt.Sprintf("%+v", event))
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (e *Engine) recordTrace(name, kind string, attrs map[string]any) {
	e.nextID++
	now := time.Now().UTC()
	e.emitOTelSpan(name, kind, attrs, now, now)
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
	_, hash := canonicalPayload(payload)
	return hash
}

// canonicalPayload returns the canonical JSON encoding of a payload and its
// sha256 hash. The bytes are what the content-addressed audit payload store
// persists, so hashing and storage can never disagree.
func canonicalPayload(payload any) ([]byte, string) {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(fmt.Sprintf("%v", payload))
	}
	sum := sha256.Sum256(b)
	return b, hex.EncodeToString(sum[:])
}

func toolInvocationKey(toolName string, payload map[string]any) string {
	return toolName + ":" + hashPayload(payload)
}

func AuditRunEventsHaveRunID(events []AuditEvent) bool {
	for _, event := range events {
		if strings.EqualFold(event.ContextType, "run") && event.RunID == "" {
			return false
		}
	}
	return true
}
