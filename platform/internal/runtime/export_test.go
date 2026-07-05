package aapruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportAndVerify(t *testing.T) {
	root, agentDir := scaffoldBAFixture(t)
	dest := filepath.Join(t.TempDir(), "export")

	manifest, err := ExportAgent(root, agentDir, dest)
	if err != nil {
		t.Fatalf("ExportAgent: %v", err)
	}
	if manifest.AgentID != "aara-business-analyst" || len(manifest.Files) < 15 {
		t.Fatalf("unexpected manifest: %s, %d files", manifest.AgentID, len(manifest.Files))
	}
	if err := VerifyExport(root, dest); err != nil {
		t.Fatalf("VerifyExport on fresh export: %v", err)
	}
	// Re-export into the same destination must be refused.
	if _, err := ExportAgent(root, agentDir, dest); err == nil {
		t.Fatal("expected refusal to overwrite an existing export")
	}
}

func TestVerifyExportDetectsTampering(t *testing.T) {
	root, agentDir := scaffoldBAFixture(t)
	dest := filepath.Join(t.TempDir(), "export")
	if _, err := ExportAgent(root, agentDir, dest); err != nil {
		t.Fatal(err)
	}

	// Modify one byte of a listed file.
	target := filepath.Join(dest, "system-prompt.md")
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, append(raw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExport(root, dest); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected hash mismatch, got: %v", err)
	}
	// Restore, then smuggle an unlisted file.
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "smuggled.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExport(root, dest); err == nil || !strings.Contains(err.Error(), "unlisted") {
		t.Fatalf("expected unlisted-file failure, got: %v", err)
	}
	// Remove the smuggled file, then delete a listed one.
	if err := os.Remove(filepath.Join(dest, "smuggled.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dest, "risk-register.md")); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExport(root, dest); err == nil {
		t.Fatal("expected failure for missing listed file")
	}
}

func TestRoundTripVerifyWritesAttestationAndCheckPasses(t *testing.T) {
	root, agentDir := scaffoldBAFixture(t)
	manifest := filepath.Join(root, "examples", "ba-agent.manifest.yaml")

	verification, err := RoundTripVerify(root, agentDir, manifest, t.TempDir())
	if err != nil {
		t.Fatalf("RoundTripVerify: %v", err)
	}
	if !verification.Identical || verification.ChecksCompared < 20 {
		t.Fatalf("unexpected verification: %+v", verification)
	}

	// The readiness check must now pass...
	report, err := RunReadiness(root, agentDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !checkPassed(report, "export-roundtrip") {
		t.Fatal("export-roundtrip must pass with a current attestation")
	}

	// ...and go stale the moment an input artifact changes.
	target := filepath.Join(agentDir, "risk-register.md")
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, append(raw, []byte("\nedited after verification\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := RunReadiness(root, agentDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if checkPassed(stale, "export-roundtrip") {
		t.Fatal("export-roundtrip must fail after inputs change (stale attestation)")
	}
}

func TestContentDigestStableAndSensitive(t *testing.T) {
	_, agentDir := scaffoldBAFixture(t)
	first, err := ContentDigest(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ContentDigest(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("digest must be stable for unchanged inputs")
	}
	// Derived outputs must not affect the digest.
	if err := os.WriteFile(filepath.Join(agentDir, "readiness-report.md"), []byte("derived"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := ContentDigest(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	if first != third {
		t.Fatal("derived report files must not change the content digest")
	}
	// Input changes must.
	if err := os.WriteFile(filepath.Join(agentDir, "system-prompt.md"), []byte("# System Prompt\n\n## Role & Objective\n\nchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fourth, err := ContentDigest(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	if first == fourth {
		t.Fatal("input changes must change the content digest")
	}
}
