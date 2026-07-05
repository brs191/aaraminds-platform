package aapruntime

import (
	"fmt"
	"strings"
)

// renderReadinessMarkdown produces the human-readable Agent Readiness Report
// (BRD v2.1 §18.3) from the schema-valid JSON report. Generated only —
// hand-authored readiness reports are invalid by definition.
func renderReadinessMarkdown(report ReadinessReport) string {
	var sb strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&sb, format, args...) }

	w("# Agent Readiness Report — %s\n\n", report.AgentID)
	w("Generated %s · rubric %s · agent version %s\n\n", report.GeneratedAt, report.RubricVersion, report.AgentVersion)
	w("## Verdict\n\n")
	w("**%s** — score %.1f/100", strings.ToUpper(report.Verdict), report.Score)
	if len(report.CriticalBlockers) > 0 {
		w(" · %d critical blocker(s)", len(report.CriticalBlockers))
	}
	w("\n\nEvery field below is populated from verifiable checks, never self-attestation.\n\n")

	w("## Agent\n\n")
	w("| Field | Value |\n|---|---|\n")
	w("| Business owner | %s |\n", report.Owners.BusinessOwner)
	w("| Technical owner | %s |\n", report.Owners.TechnicalOwner)
	w("| Autonomy level | %d |\n", report.Autonomy.Level)
	w("| Risk tier | %s |\n", report.Autonomy.RiskTier)
	w("| Justification | %s |\n\n", report.Autonomy.Justification)

	w("## Area Scores\n\n")
	w("| Area | Weight | Checks | Score |\n|---|---:|---:|---:|\n")
	for _, area := range report.Areas {
		w("| %s | %.0f | %d/%d | %.2f |\n", area.Area, area.Weight, area.ChecksPassed, area.ChecksTotal, area.Score)
	}
	w("\n")

	if len(report.CriticalBlockers) > 0 {
		w("## Critical Blockers (auto-block)\n\n")
		for _, blocker := range report.CriticalBlockers {
			w("- **%s** — %s\n  - Failing artifact: %s\n  - Required fix: %s\n", blocker.BlockerID, blocker.Description, blocker.FailingArtifact, blocker.RequiredFix)
		}
		w("\n")
	}

	w("## Failing Checks and Required Fixes\n\n")
	anyFail := false
	for _, area := range report.Areas {
		for _, check := range area.Evidence {
			if check.Result == "pass" {
				continue
			}
			anyFail = true
			w("- `%s` (%s, %s)\n  - Evidence: %s\n", check.Check, area.Area, check.Mechanism, check.EvidenceRef)
		}
	}
	if !anyFail {
		w("None — all checks passed.\n")
	}
	w("\n## Check Evidence\n\n")
	w("| Check | Result | Mechanism | Evidence |\n|---|---|---|---|\n")
	for _, area := range report.Areas {
		for _, check := range area.Evidence {
			w("| %s | %s | %s | %s |\n", check.Check, check.Result, check.Mechanism, check.EvidenceRef)
		}
	}

	w("\n## Approvals Required\n\n")
	for _, approval := range report.ApprovalsReq {
		if approval.NamedApprover != "" {
			w("- %s: %s\n", approval.Role, approval.NamedApprover)
		} else {
			w("- %s: [unassigned]\n", approval.Role)
		}
	}
	w("\nSource of truth: readiness-report.json (validated against schemas/readiness-report.schema.json).\n")
	return sb.String()
}
