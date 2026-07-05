package aapruntime

import "testing"

// TestPhase1ProofSecurityGates asserts the prompt-injection and
// memory-citation gates on the real proof run: injected escalation attempts
// are denied and audited, approvals hold in unattended mode, the manifest is
// immutable, and uncited memory writes are denied with an audited denial.
func TestPhase1ProofSecurityGates(t *testing.T) {
	root := repoRootForTest(t)
	report, err := RunPhase1Proof(root)
	if err != nil {
		t.Fatalf("RunPhase1Proof: %v", err)
	}

	if !report.InjectionToolDenied {
		t.Error("injected off-manifest tool call must be denied and audited")
	}
	if !report.InjectionApprovalEnforced {
		t.Error("injected unattended write must escalate to a hard approval, never execute")
	}
	if !report.InjectionManifestUnchanged {
		t.Error("injection scenarios must not mutate the manifest")
	}
	if !report.UncitedMemoryWriteDenied {
		t.Error("uncited memory write must be denied and must not be stored")
	}
	if !report.UncitedMemoryDenialAudited {
		t.Error("uncited memory denial must produce a memory_denied audit event")
	}

	// The denial audit event must itself sit in a valid, replayable chain.
	if !VerifyAuditChain(report.AuditEvents) {
		t.Error("audit chain must remain valid with memory_denied events included")
	}
}
