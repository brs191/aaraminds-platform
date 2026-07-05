package aapruntime

import (
	"errors"
	"fmt"
	"path/filepath"
)

// AgentIntake is the structured intake record required before any agent
// design work starts (BRD v2.1 BR-002 / AC-001). An intake that does not
// validate against schemas/agent-intake.schema.json cannot be submitted.
type AgentIntake struct {
	AgentID          string               `json:"agent_id"`
	SubmittedBy      string               `json:"submitted_by"`
	SubmittedAt      string               `json:"submitted_at"`
	BusinessProblem  string               `json:"business_problem"`
	Owners           IntakeOwners         `json:"owners"`
	Users            []string             `json:"users"`
	ExpectedOutcomes []string             `json:"expected_outcomes"`
	ExecutionIntent  string               `json:"execution_intent"`
	ProposedTools    []IntakeTool         `json:"proposed_tools"`
	DataDomains      []IntakeDataDomain   `json:"data_domains"`
	Risks            []string             `json:"risks"`
	ApprovalNeeds    string               `json:"approval_needs"`
	Classification   ClassificationInputs `json:"classification_inputs"`
}

type IntakeOwners struct {
	BusinessOwner  string `json:"business_owner"`
	TechnicalOwner string `json:"technical_owner"`
}

type IntakeTool struct {
	ToolName    string `json:"tool_name"`
	ActionType  string `json:"action_type"`
	Writes      bool   `json:"writes"`
	Description string `json:"description"`
}

type IntakeDataDomain struct {
	Domain              string `json:"domain"`
	AuthoritativeSource string `json:"authoritative_source"`
	Classification      string `json:"classification"`
}

type ClassificationInputs struct {
	ActionRisk       string `json:"action_risk"`
	DataSensitivity  string `json:"data_sensitivity"`
	Reversibility    string `json:"reversibility"`
	UserImpact       string `json:"user_impact"`
	FinancialImpact  string `json:"financial_impact"`
	ProductionImpact string `json:"production_impact"`
}

// IntakeSchemaPath returns the intake schema location under the repo root.
func IntakeSchemaPath(root string) string {
	return filepath.Join(root, "schemas", "agent-intake.schema.json")
}

// LoadIntake validates the intake file against the intake schema and decodes
// it strictly (unknown fields are rejected by loadStructuredFile).
func LoadIntake(root, path string) (AgentIntake, error) {
	var intake AgentIntake
	if path == "" {
		return intake, errors.New("intake path is required")
	}
	if err := ValidateStructuredFile(path, IntakeSchemaPath(root)); err != nil {
		return intake, fmt.Errorf("intake schema validation: %w", err)
	}
	if _, err := loadStructuredFile(path, &intake); err != nil {
		return intake, fmt.Errorf("load intake %s: %w", path, err)
	}
	if err := checkIntakeInvariants(intake); err != nil {
		return intake, fmt.Errorf("intake invariant: %w", err)
	}
	return intake, nil
}

// checkIntakeInvariants enforces cross-field rules the JSON schema cannot
// express. These are named blockers, not warnings.
func checkIntakeInvariants(intake AgentIntake) error {
	// Owners must be distinct people so accountability lines do not merge
	// silently. The same individual may hold both roles in early phases, but
	// that must be an explicit statement, not a copy-paste default.
	if intake.Owners.BusinessOwner == intake.Owners.TechnicalOwner {
		return fmt.Errorf("business_owner and technical_owner are identical (%q); if intentional, differentiate with a role suffix, e.g. %q",
			intake.Owners.BusinessOwner, intake.Owners.BusinessOwner+" (acting)")
	}
	// An agent that intends to execute must propose at least one tool.
	if intake.ExecutionIntent != "advise-only" && len(intake.ProposedTools) == 0 {
		return fmt.Errorf("execution_intent %q requires at least one proposed tool", intake.ExecutionIntent)
	}
	// Every write tool must be covered by a stated approval need.
	for _, tool := range intake.ProposedTools {
		if tool.Writes && intake.ApprovalNeeds == "" {
			return fmt.Errorf("tool %q writes but approval_needs is empty", tool.ToolName)
		}
	}
	// PII data domains force the data_sensitivity classification input up.
	for _, domain := range intake.DataDomains {
		if domain.Classification == "pii" && intake.Classification.DataSensitivity != "pii" {
			return fmt.Errorf("data domain %q is classified pii but classification_inputs.data_sensitivity is %q",
				domain.Domain, intake.Classification.DataSensitivity)
		}
	}
	return nil
}
