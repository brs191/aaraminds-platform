package aapruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// The Readiness Engine (BRD v2.1 BR-010 / AC-008) composes verifiable checks
// into a scored pass/defer/block verdict. Three invariants:
//
//  1. Every point is earned by a verifiable check — no self-attestation.
//  2. Every check result carries an evidence reference.
//  3. The engine fails closed: a rubric referencing an unknown check, or a
//     check that cannot run, is an error or a failure — never a silent pass.

// Rubric is the versioned scoring configuration loaded from
// governance/readiness-rubric.yaml.
type Rubric struct {
	RubricVersion  string       `json:"rubric_version"`
	Thresholds     Thresholds   `json:"thresholds"`
	Areas          []RubricArea `json:"areas"`
	CriticalChecks []string     `json:"critical_checks"`
}

type Thresholds struct {
	Pass  float64 `json:"pass"`
	Defer float64 `json:"defer"`
}

type RubricArea struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Weight float64  `json:"weight"`
	Checks []string `json:"checks"`
}

// ReadinessReport mirrors schemas/readiness-report.schema.json.
type ReadinessReport struct {
	AgentID          string           `json:"agent_id"`
	AgentVersion     string           `json:"agent_version"`
	RubricVersion    string           `json:"rubric_version"`
	GeneratedAt      string           `json:"generated_at"`
	Owners           ReportOwners     `json:"owners"`
	Autonomy         ReportAutonomy   `json:"autonomy"`
	Areas            []ReportArea     `json:"areas"`
	CriticalBlockers []ReportBlocker  `json:"critical_blockers"`
	Score            float64          `json:"score"`
	Verdict          string           `json:"verdict"`
	ApprovalsReq     []ReportApproval `json:"approvals_required"`
}

type ReportOwners struct {
	BusinessOwner  string `json:"business_owner"`
	TechnicalOwner string `json:"technical_owner"`
}

type ReportAutonomy struct {
	Level         int             `json:"level"`
	Justification string          `json:"justification"`
	RiskTier      string          `json:"risk_tier"`
	Signoffs      []ReportSignoff `json:"signoffs,omitempty"`
}

type ReportSignoff struct {
	Role     string `json:"role"`
	Approver string `json:"approver"`
	At       string `json:"at"`
}

type ReportArea struct {
	Area         string        `json:"area"`
	Weight       float64       `json:"weight"`
	ChecksTotal  int           `json:"checks_total"`
	ChecksPassed int           `json:"checks_passed"`
	Score        float64       `json:"score"`
	Evidence     []CheckResult `json:"evidence"`
}

type CheckResult struct {
	Check       string `json:"check"`
	Result      string `json:"result"` // pass | fail | not-applicable
	Mechanism   string `json:"mechanism"`
	EvidenceRef string `json:"evidence_ref"`
}

type ReportBlocker struct {
	BlockerID       string `json:"blocker_id"`
	Description     string `json:"description"`
	FailingArtifact string `json:"failing_artifact"`
	FailingField    string `json:"failing_field"`
	RequiredFix     string `json:"required_fix"`
}

type ReportApproval struct {
	Role          string `json:"role"`
	NamedApprover string `json:"named_approver,omitempty"`
}

// readinessContext carries loaded inputs shared across checks. Expensive
// inputs (the proof run) are computed at most once.
type readinessContext struct {
	root         string
	agentDir     string
	manifestPath string

	intake         AgentIntake
	intakeErr      error
	classification Classification

	manifest    Manifest
	manifestErr error

	signoffs []ReportSignoff

	proofOnce bool
	proof     ProofReport
	proofErr  error
}

func (rc *readinessContext) proofReport() (ProofReport, error) {
	if !rc.proofOnce {
		rc.proofOnce = true
		rc.proof, rc.proofErr = RunPhase1Proof(rc.root)
	}
	return rc.proof, rc.proofErr
}

// checkSpec couples a runner with its reporting metadata. The registry is the
// single source of truth for what a check id means; the rubric only selects
// and weights them.
type checkSpec struct {
	mechanism string
	pending   bool // defined in thresholds but not yet implemented in the harness
	run       func(rc *readinessContext) (pass bool, evidence string, fix string)
}

// LoadRubric validates and decodes the rubric configuration, then verifies
// every referenced check id exists in the registry (fail closed) and that
// weights sum to 100.
func LoadRubric(root string) (Rubric, error) {
	var rubric Rubric
	path := filepath.Join(root, "governance", "readiness-rubric.yaml")
	if err := ValidateStructuredFile(path, filepath.Join(root, "schemas", "readiness-rubric.schema.json")); err != nil {
		return rubric, fmt.Errorf("rubric schema validation: %w", err)
	}
	if _, err := loadStructuredFile(path, &rubric); err != nil {
		return rubric, fmt.Errorf("load rubric: %w", err)
	}
	total := 0.0
	seen := map[string]bool{}
	for _, area := range rubric.Areas {
		total += area.Weight
		for _, id := range area.Checks {
			if _, ok := checkRegistry[id]; !ok {
				return rubric, fmt.Errorf("rubric references unknown check %q in area %q", id, area.ID)
			}
			if seen[id] {
				return rubric, fmt.Errorf("check %q appears in more than one area", id)
			}
			seen[id] = true
		}
	}
	if total != 100 {
		return rubric, fmt.Errorf("area weights sum to %.1f, must sum to 100", total)
	}
	for _, id := range rubric.CriticalChecks {
		if !seen[id] {
			return rubric, fmt.Errorf("critical check %q is not referenced by any area", id)
		}
	}
	return rubric, nil
}

// RunReadiness executes the rubric against an agent artifact directory and
// returns the report. It does not write anything; WriteReadinessReport does.
func RunReadiness(root, agentDir, manifestPath string) (ReadinessReport, error) {
	rubric, err := LoadRubric(root)
	if err != nil {
		return ReadinessReport{}, err
	}

	rc := &readinessContext{root: root, agentDir: agentDir, manifestPath: manifestPath}

	// Load shared inputs. Failures are recorded, not fatal: the checks that
	// depend on them fail with the load error as evidence.
	rc.intake, rc.intakeErr = LoadIntake(root, filepath.Join(agentDir, "agent-intake.yaml"))
	if rc.intakeErr == nil {
		rc.classification, err = ClassifyAgent(rc.intake)
		if err != nil {
			rc.intakeErr = err
		}
	}
	if manifestPath != "" {
		rc.manifest, rc.manifestErr = LoadManifest(manifestPath)
	} else {
		rc.manifestErr = errors.New("no manifest path provided")
	}
	rc.signoffs = loadSignoffs(agentDir)

	if rc.intakeErr != nil {
		return ReadinessReport{}, fmt.Errorf("readiness requires a valid intake at %s: %w",
			filepath.Join(agentDir, "agent-intake.yaml"), rc.intakeErr)
	}

	report := ReadinessReport{
		AgentID:       rc.intake.AgentID,
		AgentVersion:  "0.1.0",
		RubricVersion: rubric.RubricVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Owners: ReportOwners{
			BusinessOwner:  rc.intake.Owners.BusinessOwner,
			TechnicalOwner: rc.intake.Owners.TechnicalOwner,
		},
		Autonomy: ReportAutonomy{
			Level:         rc.classification.AutonomyLevel,
			Justification: fmt.Sprintf("classifier: %s (risk tier %s, score %d, policy %s)", rc.classification.AutonomyName, rc.classification.RiskTier, rc.classification.RiskScore, rc.classification.MVPPolicy),
			RiskTier:      rc.classification.RiskTier,
			Signoffs:      rc.signoffs,
		},
		CriticalBlockers: []ReportBlocker{},
	}

	critical := map[string]bool{}
	for _, id := range rubric.CriticalChecks {
		critical[id] = true
	}

	total := 0.0
	for _, area := range rubric.Areas {
		reportArea := ReportArea{Area: area.Name, Weight: area.Weight}
		for _, id := range area.Checks {
			spec := checkRegistry[id]
			pass, evidence, fix := spec.run(rc)
			result := "fail"
			if pass {
				result = "pass"
			}
			reportArea.Evidence = append(reportArea.Evidence, CheckResult{
				Check:       id,
				Result:      result,
				Mechanism:   spec.mechanism,
				EvidenceRef: evidence,
			})
			reportArea.ChecksTotal++
			if pass {
				reportArea.ChecksPassed++
			} else if critical[id] {
				report.CriticalBlockers = append(report.CriticalBlockers, ReportBlocker{
					BlockerID:       "CB-" + id,
					Description:     fmt.Sprintf("critical check %q failed", id),
					FailingArtifact: evidence,
					FailingField:    id,
					RequiredFix:     fix,
				})
			}
		}
		if reportArea.ChecksTotal > 0 {
			reportArea.Score = area.Weight * float64(reportArea.ChecksPassed) / float64(reportArea.ChecksTotal)
		}
		total += reportArea.Score
		report.Areas = append(report.Areas, reportArea)
	}
	report.Score = roundTo(total, 2)

	switch {
	case len(report.CriticalBlockers) > 0:
		report.Verdict = "block"
	case report.Score >= rubric.Thresholds.Pass:
		report.Verdict = "pass"
	case report.Score >= rubric.Thresholds.Defer:
		report.Verdict = "defer"
	default:
		report.Verdict = "block"
	}

	report.ApprovalsReq = requiredApprovals(rc.classification)
	return report, nil
}

// WriteReadinessReport writes readiness-report.json and its Markdown
// rendering into the agent directory, then validates the JSON against the
// report schema — a report that fails its own schema is an engine bug.
func WriteReadinessReport(root, agentDir string, report ReadinessReport) error {
	jsonPath := filepath.Join(agentDir, "readiness-report.json")
	if err := writeJSONArtifact(jsonPath, report); err != nil {
		return err
	}
	if err := ValidateStructuredFile(jsonPath, filepath.Join(root, "schemas", "readiness-report.schema.json")); err != nil {
		return fmt.Errorf("generated readiness report failed its own schema: %w", err)
	}
	mdPath := filepath.Join(agentDir, "readiness-report.md")
	if err := os.WriteFile(mdPath, []byte(renderReadinessMarkdown(report)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", mdPath, err)
	}
	return nil
}

// ActivationGate enforces AC-008: a manifest may only hold status "active" or
// "platform-ready" when a current readiness report with verdict "pass" sits
// beside the agent's artifacts. This is called by release tooling; it is
// deliberately NOT wired into NewEngine/validateManifest so the local proof
// harness can exercise draft manifests without a readiness run.
func ActivationGate(root, agentDir, status string) error {
	if status != "active" && status != "platform-ready" {
		return nil
	}
	path := filepath.Join(agentDir, "readiness-report.json")
	var report ReadinessReport
	if _, err := loadStructuredFile(path, &report); err != nil {
		return fmt.Errorf("status %q requires a readiness report at %s: %w", status, path, err)
	}
	if err := ValidateStructuredFile(path, filepath.Join(root, "schemas", "readiness-report.schema.json")); err != nil {
		return fmt.Errorf("status %q requires a schema-valid readiness report: %w", status, err)
	}
	rubric, err := LoadRubric(root)
	if err != nil {
		return err
	}
	if report.RubricVersion != rubric.RubricVersion {
		return fmt.Errorf("readiness report was scored with rubric %s but current rubric is %s; re-run aapctl readiness",
			report.RubricVersion, rubric.RubricVersion)
	}
	if report.Verdict != "pass" {
		return fmt.Errorf("status %q requires readiness verdict \"pass\", found %q (score %.1f, %d critical blockers)",
			status, report.Verdict, report.Score, len(report.CriticalBlockers))
	}
	return nil
}

func loadSignoffs(agentDir string) []ReportSignoff {
	path := filepath.Join(agentDir, "signoffs.json")
	var signoffs []ReportSignoff
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	if _, err := loadStructuredFile(path, &signoffs); err != nil {
		return nil // malformed sign-offs are treated as absent: fail closed
	}
	return signoffs
}

func requiredApprovals(c Classification) []ReportApproval {
	roles := []string{"business-owner", "enterprise-ai-architect"}
	for _, role := range c.RequiredSignoffs {
		if !hasSignoff(roles, role) {
			roles = append(roles, role)
		}
	}
	approvals := make([]ReportApproval, 0, len(roles))
	for _, role := range roles {
		approvals = append(approvals, ReportApproval{Role: role})
	}
	return approvals
}

func roundTo(v float64, places int) float64 {
	scale := 1.0
	for i := 0; i < places; i++ {
		scale *= 10
	}
	if v >= 0 {
		return float64(int64(v*scale+0.5)) / scale
	}
	return float64(int64(v*scale-0.5)) / scale
}
