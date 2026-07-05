package aapruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkRegistry is the single source of truth for check semantics. Every
// check returns (pass, evidenceRef, requiredFix). Evidence references name
// the file, gate, or record that proves the result — a check without
// evidence is not a check.
var checkRegistry = map[string]checkSpec{

	// ---- business-scope ----
	"intake-valid": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-intake.yaml")
		if rc.intakeErr != nil {
			return false, path, "fix intake schema/invariant errors: " + rc.intakeErr.Error()
		}
		return true, path, ""
	}},
	"owners-named": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-intake.yaml")
		if rc.intake.Owners.BusinessOwner == "" || rc.intake.Owners.TechnicalOwner == "" {
			return false, path, "name both a business owner and a technical owner"
		}
		return true, path, ""
	}},
	"outcomes-stated": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-intake.yaml")
		if len(rc.intake.ExpectedOutcomes) == 0 || len(rc.intake.BusinessProblem) < 20 {
			return false, path, "state expected outcomes and a substantive business problem"
		}
		return true, path, ""
	}},

	// ---- autonomy ----
	"classification-current": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "classification.json")
		var stored Classification
		if _, err := loadStructuredFile(path, &stored); err != nil {
			return false, path, "regenerate classification.json via aapctl scaffold: " + err.Error()
		}
		if stored.AutonomyLevel != rc.classification.AutonomyLevel ||
			stored.RiskScore != rc.classification.RiskScore ||
			stored.RiskTier != rc.classification.RiskTier ||
			stored.MVPPolicy != rc.classification.MVPPolicy {
			return false, path, "classification.json has drifted from the intake; re-run aapctl scaffold -force"
		}
		return true, path, ""
	}},
	"signoffs-recorded": {mechanism: "catalog-record", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "signoffs.json")
		missing := []string{}
		for _, role := range rc.classification.RequiredSignoffs {
			found := false
			for _, s := range rc.signoffs {
				if s.Role == role && s.Approver != "" {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, role)
			}
		}
		if len(missing) > 0 {
			return false, path, "record sign-offs in signoffs.json for roles: " + strings.Join(missing, ", ")
		}
		return true, path, ""
	}},
	"manifest-valid": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "provide a loadable manifest: " + rc.manifestErr.Error()
		}
		if _, err := NewEngine(rc.root, relToRoot(rc.root, rc.manifestPath), "tool-contracts"); err != nil {
			return false, rc.manifestPath, "manifest failed full engine validation: " + err.Error()
		}
		return true, rc.manifestPath, ""
	}},
	"write-boundaries": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "manifest required to verify write boundaries"
		}
		boundaries := map[string]Boundary{}
		for _, tool := range rc.manifest.AllowedTools {
			boundaries[tool.ToolName] = tool.ApprovalBoundary
		}
		for _, tool := range rc.intake.ProposedTools {
			if !tool.Writes {
				continue
			}
			boundary, ok := boundaries[tool.ToolName]
			if !ok {
				return false, rc.manifestPath, fmt.Sprintf("write tool %q is not in the manifest allowlist", tool.ToolName)
			}
			if boundary == "none" {
				return false, rc.manifestPath, fmt.Sprintf("write tool %q has boundary \"none\"; writes require soft, hard, or blocked", tool.ToolName)
			}
		}
		return true, rc.manifestPath, ""
	}},

	// ---- contracts ----
	"contracts-exist": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		missing := []string{}
		for _, tool := range rc.intake.ProposedTools {
			path := filepath.Join(rc.root, "tool-contracts", tool.ToolName+".contract.yaml")
			if _, err := os.Stat(path); err != nil {
				missing = append(missing, tool.ToolName)
			}
		}
		if len(missing) > 0 {
			return false, filepath.Join(rc.root, "tool-contracts"), "create contracts for: " + strings.Join(missing, ", ")
		}
		return true, filepath.Join(rc.root, "tool-contracts"), ""
	}},
	"contracts-lint": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		dir := filepath.Join(rc.root, "tool-contracts")
		if _, err := LoadContractsWithSchema(dir, filepath.Join(rc.root, "schemas", "mcp-tool-contract.schema.json")); err != nil {
			return false, dir, "fix contract lint failures: " + err.Error()
		}
		return true, dir, ""
	}},
	"contracts-pinned": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "manifest required to verify contract pins"
		}
		contracts, err := LoadContractsWithSchema(filepath.Join(rc.root, "tool-contracts"), filepath.Join(rc.root, "schemas", "mcp-tool-contract.schema.json"))
		if err != nil {
			return false, rc.manifestPath, "contracts must lint before pins can be verified"
		}
		for _, tool := range rc.manifest.AllowedTools {
			contract, ok := contracts[tool.ToolName]
			if !ok {
				return false, rc.manifestPath, fmt.Sprintf("manifest pins %q but no contract exists", tool.ToolName)
			}
			if contract.ContractVersion != tool.ContractVersion {
				return false, rc.manifestPath, fmt.Sprintf("manifest pins %s@%s but contract is %s", tool.ToolName, tool.ContractVersion, contract.ContractVersion)
			}
		}
		return true, rc.manifestPath, ""
	}},
	"manifest-agent-match": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "manifest required to verify agent identity linkage"
		}
		if rc.manifest.AgentID != rc.intake.AgentID {
			return false, rc.manifestPath, fmt.Sprintf("manifest agent_id %q != intake agent_id %q; align them so the catalog, identity, and audit records reference one agent", rc.manifest.AgentID, rc.intake.AgentID)
		}
		return true, rc.manifestPath, ""
	}},

	// ---- identity ----
	"identity-valid": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-identity-spec.json")
		if err := ValidateStructuredFile(path, filepath.Join(rc.root, "schemas", "agent-identity-spec.schema.json")); err != nil {
			return false, path, "produce a schema-valid identity spec: " + err.Error()
		}
		return true, path, ""
	}},
	"identity-complete": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-identity-spec.json")
		count, err := todoCount(path)
		if err != nil {
			return false, path, err.Error()
		}
		if count > 0 {
			return false, path, fmt.Sprintf("resolve %d [TODO] markers (provisioning, rotation, retirement, sources)", count)
		}
		return true, path, ""
	}},
	"identity-scopes-match": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "agent-identity-spec.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return false, path, "identity spec unreadable"
		}
		var spec struct {
			Scopes []struct {
				Resource   string `json:"resource"`
				Permission string `json:"permission"`
			} `json:"scopes"`
		}
		if err := json.Unmarshal(raw, &spec); err != nil {
			return false, path, "identity spec unparseable"
		}
		for _, tool := range rc.intake.ProposedTools {
			want := "read"
			if tool.Writes {
				want = "write"
			}
			found := false
			for _, scope := range spec.Scopes {
				if scope.Resource == tool.ToolName && scope.Permission == want {
					found = true
					break
				}
			}
			if !found {
				return false, path, fmt.Sprintf("no %s scope for tool %q in identity spec", want, tool.ToolName)
			}
		}
		return true, path, ""
	}},

	// ---- data ----
	"evidence-contract-valid": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "data-evidence-contract.json")
		if err := ValidateStructuredFile(path, filepath.Join(rc.root, "schemas", "data-evidence-contract.schema.json")); err != nil {
			return false, path, "produce a schema-valid data/evidence contract: " + err.Error()
		}
		return true, path, ""
	}},
	"domains-mapped": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "data-evidence-contract.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return false, path, "data/evidence contract unreadable"
		}
		var contract struct {
			DataDomains []struct {
				Domain string `json:"domain"`
			} `json:"data_domains"`
		}
		if err := json.Unmarshal(raw, &contract); err != nil {
			return false, path, "data/evidence contract unparseable"
		}
		mapped := map[string]bool{}
		for _, domain := range contract.DataDomains {
			mapped[domain.Domain] = true
		}
		for _, domain := range rc.intake.DataDomains {
			if !mapped[domain.Domain] {
				return false, path, fmt.Sprintf("intake domain %q is not mapped in the evidence contract", domain.Domain)
			}
		}
		return true, path, ""
	}},
	"memory-citation-gate": {mechanism: "harness-gate", run: func(rc *readinessContext) (bool, string, string) {
		proof, err := rc.proofReport()
		if err != nil {
			return false, "platform proof harness", "proof run failed: " + err.Error()
		}
		if proof.UncitedMemoryWriteDenied && proof.UncitedMemoryDenialAudited {
			return true, "aapctl prove: memory-citation gates", ""
		}
		return false, "aapctl prove: memory-citation gates",
			"uncited memory writes must be denied and the denial audited"
	}},

	// ---- evaluation ----
	"eval-plan-sections": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		return sectionCheck(rc.agentDir, "evaluation-plan.md")
	}},
	"eval-safety-section": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "evaluation-plan.md")
		report, err := ValidateArtifactSections(path, []string{"Safety and Prompt Injection"})
		if err != nil || !report.OK() {
			return false, path, "evaluation plan must contain a non-empty Safety and Prompt Injection category"
		}
		return true, path, ""
	}},
	"eval-gate-configured": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "manifest required to verify the evaluation gate"
		}
		gate := rc.manifest.EvaluationGate
		if !gate.Required {
			return false, rc.manifestPath, "evaluation_gate.required must be true"
		}
		for _, ref := range []string{gate.BenchmarkRef, gate.ThresholdProfile} {
			if ref == "" {
				return false, rc.manifestPath, "evaluation_gate refs must be set"
			}
			if _, err := os.Stat(filepath.Join(rc.root, ref)); err != nil {
				return false, rc.manifestPath, fmt.Sprintf("evaluation_gate ref %q does not resolve", ref)
			}
		}
		return true, rc.manifestPath, ""
	}},
	"eval-runs-present": {mechanism: "eval-run", run: func(rc *readinessContext) (bool, string, string) {
		dir := filepath.Join(rc.agentDir, "eval-runs")
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			return false, dir, "record at least one eval run (schemas/eval-run.schema.json) in eval-runs/"
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if err := ValidateStructuredFile(path, filepath.Join(rc.root, "schemas", "eval-run.schema.json")); err != nil {
				return false, path, "eval run record fails schema: " + err.Error()
			}
			return true, path, ""
		}
		return false, dir, "eval-runs/ contains no schema-valid .json eval run records"
	}},

	// ---- security ----
	"asi-checklist-complete": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "security-governance-checklist.md")
		if pass, evidence, fix := sectionCheck(rc.agentDir, "security-governance-checklist.md"); !pass {
			return pass, evidence, fix
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return false, path, "checklist unreadable"
		}
		todos := strings.Count(string(raw), "Status: TODO")
		if todos > 0 {
			return false, path, fmt.Sprintf("resolve %d ASI controls still marked Status: TODO", todos)
		}
		return true, path, ""
	}},
	"proof-tool-denial": {mechanism: "harness-gate", run: func(rc *readinessContext) (bool, string, string) {
		proof, err := rc.proofReport()
		if err != nil {
			return false, "platform proof harness", "proof run failed: " + err.Error()
		}
		if proof.OffManifestToolDenied && proof.BlockedActionDenied && proof.InvalidInputDenied && proof.DenialAuditLogged {
			return true, "aapctl prove: tool-denial gates", ""
		}
		return false, "aapctl prove: tool-denial gates", "tool-denial proof gates are failing; fix the harness before any pilot"
	}},
	"proof-memory-isolation": {mechanism: "harness-gate", run: func(rc *readinessContext) (bool, string, string) {
		proof, err := rc.proofReport()
		if err != nil {
			return false, "platform proof harness", "proof run failed: " + err.Error()
		}
		if proof.MemoryLeakageReturned == 0 && proof.ExpiredMemoryReturned == 0 {
			return true, "aapctl prove: memory gates", ""
		}
		return false, "aapctl prove: memory gates",
			fmt.Sprintf("memory isolation failing: %d leaked, %d expired records returned", proof.MemoryLeakageReturned, proof.ExpiredMemoryReturned)
	}},
	"proof-audit-chain": {mechanism: "harness-gate", run: func(rc *readinessContext) (bool, string, string) {
		proof, err := rc.proofReport()
		if err != nil {
			return false, "platform proof harness", "proof run failed: " + err.Error()
		}
		if proof.AuditTrailReplayable && proof.AuditChainValid {
			return true, "aapctl prove: audit gates", ""
		}
		return false, "aapctl prove: audit gates", "audit replay/chain gates are failing"
	}},
	"prompt-injection-gate": {mechanism: "harness-gate", run: func(rc *readinessContext) (bool, string, string) {
		proof, err := rc.proofReport()
		if err != nil {
			return false, "platform proof harness", "proof run failed: " + err.Error()
		}
		if proof.InjectionToolDenied && proof.InjectionApprovalEnforced && proof.InjectionManifestUnchanged {
			return true, "aapctl prove: prompt-injection tool-escalation gates", ""
		}
		return false, "aapctl prove: prompt-injection tool-escalation gates",
			"injected tool calls must be denied, approvals must hold unattended, and the manifest must be unchanged"
	}},

	// ---- compliance ----
	"compliance-map-sections": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		return sectionCheck(rc.agentDir, "compliance-evidence-map.md")
	}},
	"compliance-complete": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, "compliance-evidence-map.md")
		count, err := todoCount(path)
		if err != nil {
			return false, path, err.Error()
		}
		if count > 0 {
			return false, path, fmt.Sprintf("resolve %d [TODO] compliance items (role confirmation, review date, jurisdiction)", count)
		}
		return true, path, ""
	}},

	// ---- export ----
	"artifacts-complete": {mechanism: "schema-validation", run: func(rc *readinessContext) (bool, string, string) {
		reports, err := ValidateArtifactDir(rc.agentDir)
		if err != nil {
			return false, rc.agentDir, "artifact validation error: " + err.Error()
		}
		for _, report := range reports {
			if !report.OK() {
				return false, report.Artifact, fmt.Sprintf("sections missing %v empty %v", report.Missing, report.Empty)
			}
		}
		for _, name := range generatedJSONArtifacts {
			if _, err := os.Stat(filepath.Join(rc.agentDir, name)); err != nil {
				return false, filepath.Join(rc.agentDir, name), "generated JSON artifact missing; re-run aapctl scaffold"
			}
		}
		return true, rc.agentDir, ""
	}},
	"telemetry-payload-mode": {mechanism: "contract-lint", run: func(rc *readinessContext) (bool, string, string) {
		if rc.manifestErr != nil {
			return false, rc.manifestPath, "manifest required to verify telemetry payload mode"
		}
		if (rc.manifest.Status == "active" || rc.manifest.Status == "platform-ready") &&
			rc.manifest.Telemetry.PayloadMode != "hash-and-reference" {
			return false, rc.manifestPath, "active/platform-ready manifests must use hash-and-reference payload mode"
		}
		return true, rc.manifestPath, ""
	}},
	"export-roundtrip": {mechanism: "export-roundtrip", run: func(rc *readinessContext) (bool, string, string) {
		path := filepath.Join(rc.agentDir, exportVerificationName)
		if err := ValidateStructuredFile(path, filepath.Join(rc.root, "schemas", "export-verification.schema.json")); err != nil {
			return false, path, "run aapctl export -verify to produce a round-trip verification: " + err.Error()
		}
		var verification ExportVerification
		if _, err := loadStructuredFile(path, &verification); err != nil {
			return false, path, "verification unreadable: " + err.Error()
		}
		if !verification.Identical {
			return false, path, "recorded round-trip was not identical; investigate and re-run aapctl export -verify"
		}
		digest, err := ContentDigest(rc.agentDir)
		if err != nil {
			return false, path, "content digest failed: " + err.Error()
		}
		if digest != verification.ContentDigest {
			return false, path, "artifacts changed since the last verified round-trip; re-run aapctl export -verify"
		}
		return true, path, ""
	}},
}

func sectionCheck(agentDir, artifact string) (bool, string, string) {
	path := filepath.Join(agentDir, artifact)
	required, ok := ArtifactSections[artifact]
	if !ok {
		return false, path, "no section registry entry for " + artifact
	}
	report, err := ValidateArtifactSections(path, required)
	if err != nil {
		return false, path, "artifact unreadable: " + err.Error()
	}
	if !report.OK() {
		return false, path, fmt.Sprintf("sections missing %v empty %v", report.Missing, report.Empty)
	}
	return true, path, ""
}

func todoCount(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("unreadable: %w", err)
	}
	return strings.Count(string(raw), "[TODO"), nil
}

// relToRoot converts an absolute path under root back to a root-relative
// path, because NewEngine joins root with the manifest path itself.
func relToRoot(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
