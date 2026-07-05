package aapruntime

import "fmt"

// Classification is the deterministic output of the autonomy and risk
// classifier (BRD v2.1 BR-003 / AC-002). No model calls: same intake in,
// same classification out.
type Classification struct {
	AutonomyLevel    int      `json:"autonomy_level"`
	AutonomyName     string   `json:"autonomy_name"`
	RiskScore        int      `json:"risk_score"`
	RiskTier         string   `json:"risk_tier"`
	RequiredSignoffs []string `json:"required_signoffs"`
	MVPPolicy        string   `json:"mvp_policy"`
	Rationale        []string `json:"rationale"`
}

var autonomyNames = map[int]string{
	1: "Advisory",
	2: "Drafting",
	3: "Approval-Gated Execution",
	4: "Bounded Execution",
	5: "Autonomous Execution",
}

var intentLevels = map[string]int{
	"advise-only":           1,
	"draft-outputs":         2,
	"execute-with-approval": 3,
	"execute-bounded":       4,
	"execute-autonomous":    5,
}

// ClassifyAgent derives autonomy level, risk tier, and required sign-offs
// from intake classification inputs using a fixed rules table.
//
// Risk scoring: low/reversible/public = 0 ... high/irreversible = 2, with
// data sensitivity scoring up to 3 (pii). Tiers: 0-2 low, 3-5 medium,
// 6-8 high, 9+ critical. Level caps: high tier caps at 4, critical at 3.
func ClassifyAgent(intake AgentIntake) (Classification, error) {
	level, ok := intentLevels[intake.ExecutionIntent]
	if !ok {
		return Classification{}, fmt.Errorf("unknown execution_intent %q", intake.ExecutionIntent)
	}
	c := Classification{AutonomyLevel: level, RequiredSignoffs: []string{}}
	c.Rationale = append(c.Rationale,
		fmt.Sprintf("execution_intent %q maps to level %d (%s)", intake.ExecutionIntent, level, autonomyNames[level]))

	c.RiskScore = riskScore(intake.Classification)
	c.RiskTier = riskTier(c.RiskScore)
	c.Rationale = append(c.Rationale,
		fmt.Sprintf("risk score %d from classification inputs -> tier %s", c.RiskScore, c.RiskTier))

	// Risk-tier caps. A critical-risk agent never exceeds approval-gated
	// execution; a high-risk agent never runs fully autonomous.
	cap := 5
	switch c.RiskTier {
	case "critical":
		cap = 3
	case "high":
		cap = 4
	}
	if c.AutonomyLevel > cap {
		c.Rationale = append(c.Rationale,
			fmt.Sprintf("level capped from %d to %d by risk tier %s", c.AutonomyLevel, cap, c.RiskTier))
		c.AutonomyLevel = cap
	}
	c.AutonomyName = autonomyNames[c.AutonomyLevel]

	// Write tools always require contract-level approval boundaries,
	// independent of autonomy level.
	for _, tool := range intake.ProposedTools {
		if tool.Writes {
			c.Rationale = append(c.Rationale,
				fmt.Sprintf("tool %q writes: contract must set approval_boundary soft, hard, or blocked", tool.ToolName))
		}
	}

	// Sign-off rules.
	if c.AutonomyLevel >= 4 {
		c.RequiredSignoffs = append(c.RequiredSignoffs, "business-owner", "security-reviewer", "operations-owner")
	}
	if (c.RiskTier == "high" || c.RiskTier == "critical") && !hasSignoff(c.RequiredSignoffs, "security-reviewer") {
		c.RequiredSignoffs = append(c.RequiredSignoffs, "security-reviewer")
	}
	if intake.Classification.DataSensitivity == "pii" && !hasSignoff(c.RequiredSignoffs, "compliance-lead") {
		c.RequiredSignoffs = append(c.RequiredSignoffs, "compliance-lead")
	}

	// MVP policy per BRD v2.1 §17: levels 1-3 supported; 4 requires explicit
	// sign-off; 5 is out of scope until platform maturity is proven.
	switch {
	case c.AutonomyLevel <= 3:
		c.MVPPolicy = "allowed"
	case c.AutonomyLevel == 4:
		c.MVPPolicy = "requires-signoff"
	default:
		c.MVPPolicy = "out-of-scope"
	}
	return c, nil
}

func riskScore(in ClassificationInputs) int {
	score := lmh(in.ActionRisk) + lmh(in.UserImpact) + lmh(in.FinancialImpact) + lmh(in.ProductionImpact)
	switch in.DataSensitivity {
	case "internal":
		score++
	case "client-confidential":
		score += 2
	case "pii":
		score += 3
	}
	switch in.Reversibility {
	case "partially-reversible":
		score++
	case "irreversible":
		score += 2
	}
	return score
}

func lmh(v string) int {
	switch v {
	case "medium":
		return 1
	case "high":
		return 2
	default:
		return 0
	}
}

func riskTier(score int) string {
	switch {
	case score <= 2:
		return "low"
	case score <= 5:
		return "medium"
	case score <= 8:
		return "high"
	default:
		return "critical"
	}
}

func hasSignoff(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
