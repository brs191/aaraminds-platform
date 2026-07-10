package aapruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllPacksValid(t *testing.T) {
	root := repoRootForTest(t)
	if _, err := os.Stat(filepath.Join(root, "packs")); err != nil {
		t.Skip("packs/ not present")
	}
	packs, err := LoadAllPacks(root)
	if err != nil {
		t.Fatalf("LoadAllPacks: %v", err)
	}
	if len(packs) < 3 {
		t.Fatalf("expected at least 3 packs, got %d", len(packs))
	}
	for _, p := range packs {
		if p.PackID == "" || len(p.Members) == 0 {
			t.Errorf("pack %q malformed", p.PackID)
		}
	}
}

func TestPackReadinessRollupReadsRealReports(t *testing.T) {
	root := repoRootForTest(t)
	path := filepath.Join(root, "packs", "data-engineering-pack.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip("data-engineering-pack not present")
	}
	pack, err := LoadPack(root, path)
	if err != nil {
		t.Fatal(err)
	}
	card, err := RunPackReadiness(root, pack)
	if err != nil {
		t.Fatal(err)
	}
	// The three certified members must resolve to real, current reports (their
	// readiness reports exist and were scored under the current rubric).
	if card.Counts.CertifiedCurrent+card.Counts.CertifiedStale < 3 {
		t.Fatalf("expected >=3 certified members resolved, got current=%d stale=%d",
			card.Counts.CertifiedCurrent, card.Counts.CertifiedStale)
	}
	// No member declared certified may be missing its report — the pack must be honest.
	if card.Counts.ReportMissing > 0 {
		t.Errorf("pack claims %d certified members with no readiness report", card.Counts.ReportMissing)
	}
	// The certified average must be a plausible score when any are current.
	if card.Counts.CertifiedCurrent > 0 {
		if card.CertifiedAvgScore == nil || *card.CertifiedAvgScore <= 0 || *card.CertifiedAvgScore > 100 {
			t.Errorf("implausible certified avg: %v", card.CertifiedAvgScore)
		}
	}
}

func TestPackReportMissingIsHonest(t *testing.T) {
	root := repoRootForTest(t)
	// A pack claiming a certified member that has no readiness report must be
	// reported as report-missing, never counted as certified.
	pack := Pack{
		PackID: "test-pack", Name: "T", Domain: "d", Timeline: "now",
		Description: "a description long enough to satisfy the schema minimum length",
		Members: []PackMember{
			{AgentID: "does-not-exist-agent", Status: "certified"},
		},
	}
	card, err := RunPackReadiness(root, pack)
	if err != nil {
		t.Fatal(err)
	}
	if card.Counts.ReportMissing != 1 || card.Counts.CertifiedCurrent != 0 {
		t.Fatalf("a certified member without a report must be report-missing, got %+v", card.Counts)
	}
	if card.CertifiedAvgScore != nil {
		t.Error("no current-certified members means no average")
	}
}

func TestWritePackScorecardSelfValidates(t *testing.T) {
	root := repoRootForTest(t)
	path := filepath.Join(root, "packs", "data-engineering-pack.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip("data-engineering-pack not present")
	}
	pack, err := LoadPack(root, path)
	if err != nil {
		t.Fatal(err)
	}
	card, err := RunPackReadiness(root, pack)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WritePackScorecard(root, out, card); err != nil {
		t.Fatalf("WritePackScorecard (includes schema self-check): %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, card.PackID+"-scorecard.md")); err != nil {
		t.Errorf("markdown scorecard not written: %v", err)
	}
}
