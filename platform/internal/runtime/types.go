package aapruntime

import (
	"encoding/json"
	"time"
)

type Boundary string

const (
	BoundaryNone    Boundary = "none"
	BoundarySoft    Boundary = "soft"
	BoundaryHard    Boundary = "hard"
	BoundaryBlocked Boundary = "blocked"
)

type RuntimeMode string

const (
	RuntimeInteractive RuntimeMode = "interactive"
	RuntimeUnattended  RuntimeMode = "unattended"
)

type Manifest struct {
	AgentID            string             `json:"agent_id"`
	ManifestVersion    string             `json:"manifest_version"`
	Owner              string             `json:"owner"`
	Runtime            string             `json:"runtime"`
	Status             string             `json:"status"`
	AllowedSkills      []ManifestSkill    `json:"allowed_skills"`
	AllowedTools       []ManifestTool     `json:"allowed_tools"`
	Memory             ManifestMemory     `json:"memory"`
	ApprovalBoundaries ApprovalBoundaries `json:"approval_boundaries"`
	Telemetry          ManifestTelemetry  `json:"telemetry"`
	EvaluationGate     EvaluationGate     `json:"evaluation_gate"`
}

type ManifestSkill struct {
	SkillID      string `json:"skill_id"`
	SkillVersion string `json:"skill_version"`
	SourcePath   string `json:"source_path"`
}

type ManifestTool struct {
	ToolName         string   `json:"tool_name"`
	ContractVersion  string   `json:"contract_version"`
	ApprovalBoundary Boundary `json:"approval_boundary"`
}

type ManifestMemory struct {
	Enabled                bool     `json:"enabled"`
	Scope                  string   `json:"scope"`
	AllowedClassifications []string `json:"allowed_classifications"`
	PIIAllowed             bool     `json:"pii_allowed"`
}

type ApprovalBoundaries struct {
	Default           Boundary `json:"default"`
	BlockedActionsRef string   `json:"blocked_actions_ref"`
}

type ManifestTelemetry struct {
	OTELEnabled     bool   `json:"otel_enabled"`
	CostAttribution bool   `json:"cost_attribution"`
	PayloadMode     string `json:"payload_mode"`
}

type EvaluationGate struct {
	Required         bool   `json:"required"`
	BenchmarkRef     string `json:"benchmark_ref"`
	ThresholdProfile string `json:"threshold_profile"`
}

type ToolContract struct {
	ToolName            string                 `json:"tool_name"`
	ContractVersion     string                 `json:"contract_version"`
	ActionType          string                 `json:"action_type"`
	Purpose             string                 `json:"purpose"`
	InputSchema         map[string]any         `json:"input_schema"`
	OutputSchema        map[string]any         `json:"output_schema"`
	PermissionsRequired []string               `json:"permissions_required"`
	ApprovalBoundary    Boundary               `json:"approval_boundary"`
	DataClassification  DataClassification     `json:"data_classification"`
	FailureModes        []FailureMode          `json:"failure_modes"`
	TimeoutMS           int                    `json:"timeout_ms"`
	TimeoutClass        string                 `json:"timeout_class"`
	RetryPolicy         RetryPolicy            `json:"retry_policy"`
	AuditEventSchema    map[string]any         `json:"audit_event_schema"`
	ExampleInvocation   map[string]interface{} `json:"example_invocation"`
}

type DataClassification struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type FailureMode struct {
	Code            string `json:"code"`
	Meaning         string `json:"meaning"`
	Retryable       bool   `json:"retryable"`
	SafeUserMessage string `json:"safe_user_message"`
}

type RetryPolicy struct {
	MaxAttempts int    `json:"max_attempts"`
	Backoff     string `json:"backoff"`
}

type BlockedActions struct {
	Version                     string   `json:"version"`
	BlockedActions              []string `json:"blocked_actions"`
	ClassifiedActions           []string `json:"classified_actions"`
	DefaultUnclassifiedBoundary Boundary `json:"default_unclassified_boundary"`
	MissingContractBoundary     Boundary `json:"missing_contract_boundary"`
}

type RunContext struct {
	RunID           string `json:"run_id"`
	AgentID         string `json:"agent_id"`
	ManifestVersion string `json:"manifest_version"`
	EngagementID    string `json:"engagement_id"`
	UserID          string `json:"user_id"`
	TenantNamespace string `json:"tenant_namespace"`
}

type AuditEvent struct {
	AuditEventID  string    `json:"audit_event_id"`
	EventType     string    `json:"event_type"`
	ActorType     string    `json:"actor_type"`
	ActorID       string    `json:"actor_id"`
	ContextType   string    `json:"context_type"`
	ContextID     string    `json:"context_id"`
	RunID         string    `json:"run_id,omitempty"`
	PayloadRef    string    `json:"payload_ref"`
	PayloadHash   string    `json:"payload_hash"`
	PrevEventHash string    `json:"prev_event_hash"`
	Timestamp     time.Time `json:"timestamp"`
}

// ApprovalRequest mirrors schemas/approval-request.schema.json. A request is
// created when a tool invocation hits a soft or hard approval boundary,
// resolved by a named approver, and consumed at most once by a matching
// re-invocation.
type ApprovalRequest struct {
	ApprovalRequestID string      `json:"approval_request_id"`
	RunID             string      `json:"run_id"`
	ToolInvocationID  string      `json:"tool_invocation_id"`
	ApprovalBoundary  Boundary    `json:"approval_boundary"`
	RequestedAction   string      `json:"requested_action"`
	RiskSummary       string      `json:"risk_summary"`
	RuntimeMode       RuntimeMode `json:"runtime_mode"`
	Status            string      `json:"status"`
	ApproverID        string      `json:"approver_id,omitempty"`
	DecisionReason    string      `json:"decision_reason,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	DecidedAt         *time.Time  `json:"decided_at,omitempty"`

	consumed bool
}

type TraceSpan struct {
	TraceID   string         `json:"trace_id"`
	SpanID    string         `json:"span_id"`
	RunID     string         `json:"run_id"`
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	Attrs     map[string]any `json:"attrs"`
}

type ToolDecision struct {
	ToolName          string   `json:"tool_name"`
	Outcome           string   `json:"outcome"`
	ApprovalBoundary  Boundary `json:"approval_boundary"`
	Reason            string   `json:"reason"`
	AuditEventID      string   `json:"audit_event_id,omitempty"`
	ApprovalRequestID string   `json:"approval_request_id,omitempty"`
}

type MemoryRecord struct {
	MemoryID       string         `json:"memory_id"`
	AgentID        string         `json:"agent_id"`
	EngagementID   string         `json:"engagement_id"`
	Classification string         `json:"classification"`
	ContentRef     string         `json:"content_ref"`
	ContentHash    string         `json:"content_hash"`
	SourceCitation SourceCitation `json:"source_citation"`
	CreatedAt      time.Time      `json:"created_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
	Status         string         `json:"status"`
}

type SourceCitation struct {
	RunID     string `json:"run_id"`
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
	SourceRef string `json:"source_ref"`
}

type ProofReport struct {
	RunContext                  RunContext                 `json:"run_context"`
	ValidManifestStarted        bool                       `json:"valid_manifest_started"`
	AllowedToolExecuted         bool                       `json:"allowed_tool_executed"`
	ToolOutputAccepted          bool                       `json:"tool_output_accepted"`
	OffManifestToolDenied       bool                       `json:"off_manifest_tool_denied"`
	BlockedActionDenied         bool                       `json:"blocked_action_denied"`
	InvalidInputDenied          bool                       `json:"invalid_input_denied"`
	OutputSchemaViolationDenied bool                       `json:"output_schema_violation_denied"`
	TimeoutViolationDenied      bool                       `json:"timeout_violation_denied"`
	RetryPolicyViolationDenied  bool                       `json:"retry_policy_violation_denied"`
	DuplicateResultDenied       bool                       `json:"duplicate_result_denied"`
	DenialAuditLogged           bool                       `json:"denial_audit_logged"`
	SoftUnattendedEscalated     bool                       `json:"soft_unattended_escalated"`
	ApprovalGrantAudited        bool                       `json:"approval_grant_audited"`
	ApprovedInvocationExecuted  bool                       `json:"approved_invocation_executed"`
	ApprovalGrantSingleUse      bool                       `json:"approval_grant_single_use"`
	AuditTrailReplayable        bool                       `json:"audit_trail_replayable"`
	AuditChainValid             bool                       `json:"audit_chain_valid"`
	MemoryLeakageReturned       int                        `json:"memory_leakage_returned"`
	ExpiredMemoryReturned       int                        `json:"expired_memory_returned"`
	RunScopedAuditsHaveRunID    bool                       `json:"run_scoped_audits_have_run_id"`
	TraceSpanCount              int                        `json:"trace_span_count"`
	AuditEvents                 []AuditEvent               `json:"audit_events"`
	ApprovalRequests            []ApprovalRequest          `json:"approval_requests"`
	AuditPayloads               map[string]json.RawMessage `json:"audit_payloads"`
	ToolDecisions               []ToolDecision             `json:"tool_decisions"`
}
