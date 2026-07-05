package aapruntime

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestASITitlesOfficial pins the OWASP ASI01–ASI10 titles to the official
// 2026 taxonomy. If OWASP revises the list, this test is the tripwire — it
// fails until asi.go is updated deliberately, not silently.
func TestASITitlesOfficial(t *testing.T) {
	want := []string{
		"ASI01 Agent Goal Hijack",
		"ASI02 Tool Misuse & Exploitation",
		"ASI03 Agent Identity & Privilege Abuse",
		"ASI04 Agentic Supply Chain Compromise",
		"ASI05 Unexpected Code Execution",
		"ASI06 Memory & Context Poisoning",
		"ASI07 Insecure Inter-Agent Communication",
		"ASI08 Cascading Agent Failures",
		"ASI09 Human-Agent Trust Exploitation",
		"ASI10 Rogue Agents",
	}
	if len(ASITitles) != len(want) {
		t.Fatalf("expected %d ASI titles, got %d", len(want), len(ASITitles))
	}
	for i, title := range want {
		if ASITitles[i] != title {
			t.Errorf("ASI title %d: got %q, want %q", i+1, ASITitles[i], title)
		}
	}
}

// TestSecurityChecklistUsesSharedTitles guards against the two-source drift
// that caused the original bug: the section registry must consume ASITitles,
// so a scaffolded checklist and the validator can never diverge.
func TestSecurityChecklistUsesSharedTitles(t *testing.T) {
	sections := ArtifactSections["security-governance-checklist.md"]
	for _, title := range ASITitles {
		found := false
		for _, section := range sections {
			if section == title {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("security checklist registry missing shared ASI title %q", title)
		}
	}
}

// TestScaffoldedChecklistHeadingsMatchTitles renders the BA scaffold and
// confirms every official ASI title appears as a real "## " heading. This ties
// the template text to the shared constant end to end.
func TestScaffoldedChecklistHeadingsMatchTitles(t *testing.T) {
	_, dir := scaffoldBAFixture(t)
	rawBytes, err := os.ReadFile(filepath.Join(dir, "security-governance-checklist.md"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(rawBytes)
	for _, title := range ASITitles {
		heading := "## " + title
		matched, _ := regexp.MatchString("(?m)^"+regexp.QuoteMeta(heading)+"\\s*$", raw)
		if !matched {
			t.Errorf("scaffolded checklist missing heading %q", heading)
		}
	}
	// The stale [VERIFY] disclaimer must be gone now that titles are official.
	if strings.Contains(raw, "[VERIFY official ASI titles") {
		t.Error("stale ASI [VERIFY] disclaimer still present in generated checklist")
	}
}
