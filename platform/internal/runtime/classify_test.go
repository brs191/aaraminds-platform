package aapruntime

import (
	"strings"
	"testing"
)

func intakeWith(intent string, in ClassificationInputs, tools ...IntakeTool) AgentIntake {
	return AgentIntake{
		AgentID:         "test-agent",
		ExecutionIntent: intent,
		ProposedTools:   tools,
		Classification:  in,
	}
}

func lowInputs() ClassificationInputs {
	return ClassificationInputs{
		ActionRisk:       "low",
		DataSensitivity:  "public",
		Reversibility:    "reversible",
		UserImpact:       "low",
		FinancialImpact:  "low",
		ProductionImpact: "low",
	}
}

func TestClassifyAdvisoryLowRisk(t *testing.T) {
	c, err := ClassifyAgent(intakeWith("advise-only", lowInputs()))
	if err != nil {
		t.Fatal(err)
	}
	if c.AutonomyLevel != 1 || c.RiskTier != "low" || c.MVPPolicy != "allowed" {
		t.Fatalf("got %+v", c)
	}
	if len(c.RequiredSignoffs) != 0 {
		t.Fatalf("expected no signoffs, got %v", c.RequiredSignoffs)
	}
}

func TestClassifyDraftingClientConfidential(t *testing.T) {
	in := lowInputs()
	in.DataSensitivity = "client-confidential"
	in.UserImpact = "medium"
	c, err := ClassifyAgent(intakeWith("draft-outputs", in))
	if err != nil {
		t.Fatal(err)
	}
	// score: 2 (cc) + 1 (medium user impact) = 3 -> medium
	if c.AutonomyLevel != 2 || c.RiskTier != "medium" {
		t.Fatalf("got level %d tier %s", c.AutonomyLevel, c.RiskTier)
	}
}

func TestClassifyCriticalCapsLevelAtThree(t *testing.T) {
	in := ClassificationInputs{
		ActionRisk:       "high",
		DataSensitivity:  "pii",
		Reversibility:    "irreversible",
		UserImpact:       "high",
		FinancialImpact:  "high",
		ProductionImpact: "high",
	}
	// score: 2+3+2+2+2+2 = 13 -> critical
	c, err := ClassifyAgent(intakeWith("execute-autonomous", in))
	if err != nil {
		t.Fatal(err)
	}
	if c.RiskTier != "critical" {
		t.Fatalf("tier = %s", c.RiskTier)
	}
	if c.AutonomyLevel != 3 {
		t.Fatalf("critical risk must cap level at 3, got %d", c.AutonomyLevel)
	}
	if !containsString(c.RequiredSignoffs, "security-reviewer") || !containsString(c.RequiredSignoffs, "compliance-lead") {
		t.Fatalf("signoffs = %v", c.RequiredSignoffs)
	}
	if c.MVPPolicy != "allowed" {
		t.Fatalf("capped level 3 should be allowed, got %s", c.MVPPolicy)
	}
}

func TestClassifyHighTierCapsAtFourAndRequiresSignoff(t *testing.T) {
	in := ClassificationInputs{
		ActionRisk:       "high",
		DataSensitivity:  "client-confidential",
		Reversibility:    "partially-reversible",
		UserImpact:       "medium",
		FinancialImpact:  "low",
		ProductionImpact: "medium",
	}
	// score: 2+2+1+1+0+1 = 7 -> high
	c, err := ClassifyAgent(intakeWith("execute-autonomous", in))
	if err != nil {
		t.Fatal(err)
	}
	if c.RiskTier != "high" || c.AutonomyLevel != 4 {
		t.Fatalf("got tier %s level %d", c.RiskTier, c.AutonomyLevel)
	}
	if c.MVPPolicy != "requires-signoff" {
		t.Fatalf("level 4 must require signoff, got %s", c.MVPPolicy)
	}
	for _, role := range []string{"business-owner", "security-reviewer", "operations-owner"} {
		if !containsString(c.RequiredSignoffs, role) {
			t.Fatalf("missing signoff %s in %v", role, c.RequiredSignoffs)
		}
	}
}

func TestClassifyLevelFiveOutOfScope(t *testing.T) {
	c, err := ClassifyAgent(intakeWith("execute-autonomous", lowInputs()))
	if err != nil {
		t.Fatal(err)
	}
	if c.AutonomyLevel != 5 || c.MVPPolicy != "out-of-scope" {
		t.Fatalf("got level %d policy %s", c.AutonomyLevel, c.MVPPolicy)
	}
}

func TestClassifyWriteToolRationale(t *testing.T) {
	c, err := ClassifyAgent(intakeWith("draft-outputs", lowInputs(),
		IntakeTool{ToolName: "create_draft", ActionType: "draft_create", Writes: true, Description: "d"}))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range c.Rationale {
		if strings.Contains(r, "create_draft") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected write-tool rationale, got %v", c.Rationale)
	}
}

func TestClassifyWriteToolForcesRiskFloor(t *testing.T) {
	// Self-attested "low" action risk must be floored to medium when the
	// intake itself declares write tools.
	c, err := ClassifyAgent(intakeWith("draft-outputs", lowInputs(),
		IntakeTool{ToolName: "create_draft", ActionType: "draft_create", Writes: true, Description: "d"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.RiskScore != 1 {
		t.Fatalf("expected risk score 1 (floored medium action risk), got %d", c.RiskScore)
	}
	floored := false
	for _, r := range c.Rationale {
		if strings.Contains(r, "floored") {
			floored = true
		}
	}
	if !floored {
		t.Fatalf("expected floor rationale, got %v", c.Rationale)
	}
	// Read-only agent with identical inputs keeps score 0.
	readonly, err := ClassifyAgent(intakeWith("draft-outputs", lowInputs(),
		IntakeTool{ToolName: "read_thing", ActionType: "thing_read", Writes: false, Description: "d"}))
	if err != nil {
		t.Fatal(err)
	}
	if readonly.RiskScore != 0 {
		t.Fatalf("read-only agent should score 0, got %d", readonly.RiskScore)
	}
}

func TestClassifyUnknownIntent(t *testing.T) {
	if _, err := ClassifyAgent(intakeWith("yolo", lowInputs())); err == nil {
		t.Fatal("expected error for unknown intent")
	}
}

func TestClassifyDeterministic(t *testing.T) {
	intake := intakeWith("execute-with-approval", ClassificationInputs{
		ActionRisk: "medium", DataSensitivity: "internal", Reversibility: "reversible",
		UserImpact: "medium", FinancialImpact: "low", ProductionImpact: "low",
	})
	first, err := ClassifyAgent(intake)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		next, err := ClassifyAgent(intake)
		if err != nil {
			t.Fatal(err)
		}
		if next.AutonomyLevel != first.AutonomyLevel || next.RiskScore != first.RiskScore || next.RiskTier != first.RiskTier {
			t.Fatalf("non-deterministic classification: %+v vs %+v", first, next)
		}
	}
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
