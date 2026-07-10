package aapruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// A pack is a curated collection of individually-certified agents sharing a
// domain. It is an organizational grouping, not a runtime entity: each member
// remains its own manifest and readiness verdict. The pack rollup reads each
// member's committed readiness report — it never self-attests a member as
// certified without the artifact (a "certified" member with no report is
// reported as report-missing, an honesty failure the deck must not hide).

// Pack mirrors schemas/pack.schema.json.
type Pack struct {
	PackID              string       `json:"pack_id"`
	Name                string       `json:"name"`
	Domain              string       `json:"domain"`
	Timeline            string       `json:"timeline"`
	Description         string       `json:"description"`
	Members             []PackMember `json:"members"`
	SharedToolContracts []string     `json:"shared_tool_contracts,omitempty"`
}

type PackMember struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Note    string `json:"note,omitempty"`
}

// PackScorecard mirrors schemas/pack-scorecard.schema.json.
type PackScorecard struct {
	PackID            string             `json:"pack_id"`
	Name              string             `json:"name"`
	Domain            string             `json:"domain"`
	Timeline          string             `json:"timeline"`
	GeneratedAt       string             `json:"generated_at"`
	RubricVersion     string             `json:"rubric_version"`
	MemberCount       int                `json:"member_count"`
	CertifiedAvgScore *float64           `json:"certified_avg_score"`
	Counts            PackCounts         `json:"counts"`
	Members           []PackMemberResult `json:"members"`
}

type PackCounts struct {
	CertifiedCurrent int `json:"certified_current"`
	CertifiedStale   int `json:"certified_stale"`
	ReportMissing    int `json:"report_missing"`
	Defined          int `json:"defined"`
	Planned          int `json:"planned"`
}

type PackMemberResult struct {
	AgentID             string   `json:"agent_id"`
	DeclaredStatus      string   `json:"declared_status"`
	State               string   `json:"state"`
	Score               *float64 `json:"score,omitempty"`
	Verdict             string   `json:"verdict,omitempty"`
	ReportRubricVersion string   `json:"report_rubric_version,omitempty"`
	Note                string   `json:"note,omitempty"`
}

// LoadPack validates a pack manifest against the schema and decodes it.
func LoadPack(root, path string) (Pack, error) {
	var pack Pack
	if err := ValidateStructuredFile(path, filepath.Join(root, "schemas", "pack.schema.json")); err != nil {
		return pack, fmt.Errorf("pack schema validation: %w", err)
	}
	if _, err := loadStructuredFile(path, &pack); err != nil {
		return pack, fmt.Errorf("load pack %s: %w", path, err)
	}
	return pack, nil
}

// RunPackReadiness rolls up member readiness for a pack. For each member
// declared "certified" it reads agents/<id>/readiness-report.json and records
// the verified score/verdict; a missing report on a certified member is a
// report-missing state (never silently treated as certified). Members declared
// defined/planned are reported as-is. The certified average covers only members
// with a current-rubric passing/deferring report.
func RunPackReadiness(root string, pack Pack) (PackScorecard, error) {
	rubric, err := LoadRubric(root)
	if err != nil {
		return PackScorecard{}, err
	}
	card := PackScorecard{
		PackID:        pack.PackID,
		Name:          pack.Name,
		Domain:        pack.Domain,
		Timeline:      pack.Timeline,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		RubricVersion: rubric.RubricVersion,
		MemberCount:   len(pack.Members),
	}
	var certifiedSum float64
	var certifiedN int
	for _, member := range pack.Members {
		result := PackMemberResult{
			AgentID:        member.AgentID,
			DeclaredStatus: member.Status,
			Note:           member.Note,
		}
		switch member.Status {
		case "planned":
			result.State = "planned"
			card.Counts.Planned++
		case "defined":
			result.State = "defined"
			card.Counts.Defined++
		case "certified":
			reportPath := filepath.Join(root, "agents", member.AgentID, "readiness-report.json")
			var report ReadinessReport
			if _, err := loadStructuredFile(reportPath, &report); err != nil {
				// Declared certified but no readable report — an honesty
				// failure, surfaced, not hidden.
				result.State = "report-missing"
				card.Counts.ReportMissing++
				break
			}
			score := report.Score
			result.Score = &score
			result.Verdict = report.Verdict
			result.ReportRubricVersion = report.RubricVersion
			if report.RubricVersion != rubric.RubricVersion {
				result.State = "certified-stale"
				card.Counts.CertifiedStale++
			} else {
				result.State = "certified-current"
				card.Counts.CertifiedCurrent++
				certifiedSum += score
				certifiedN++
			}
		}
		card.Members = append(card.Members, result)
	}
	if certifiedN > 0 {
		avg := roundTo(certifiedSum/float64(certifiedN), 1)
		card.CertifiedAvgScore = &avg
	}
	return card, nil
}

// WritePackScorecard writes <pack_id>-scorecard.json + .md into outDir and
// self-validates the JSON against the scorecard schema.
func WritePackScorecard(root, outDir string, card PackScorecard) error {
	jsonPath := filepath.Join(outDir, card.PackID+"-scorecard.json")
	if err := writeJSONArtifact(jsonPath, card); err != nil {
		return err
	}
	if err := ValidateStructuredFile(jsonPath, filepath.Join(root, "schemas", "pack-scorecard.schema.json")); err != nil {
		return fmt.Errorf("generated pack scorecard failed its own schema: %w", err)
	}
	mdPath := filepath.Join(outDir, card.PackID+"-scorecard.md")
	if err := os.WriteFile(mdPath, []byte(renderPackScorecard(card)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", mdPath, err)
	}
	return nil
}

// LoadAllPacks loads every pack manifest in the packs/ directory, sorted by id.
func LoadAllPacks(root string) ([]Pack, error) {
	dir := filepath.Join(root, "packs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read packs dir: %w", err)
	}
	var packs []Pack
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		pack, err := LoadPack(root, filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	sort.Slice(packs, func(i, j int) bool { return packs[i].PackID < packs[j].PackID })
	return packs, nil
}

func renderPackScorecard(card PackScorecard) string {
	var sb strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&sb, format, args...) }
	w("# Pack Scorecard — %s\n\n", card.Name)
	w("Domain: %s · Timeline: %s · Generated %s · rubric %s\n\n", card.Domain, card.Timeline, card.GeneratedAt, card.RubricVersion)
	avg := "n/a"
	if card.CertifiedAvgScore != nil {
		avg = fmt.Sprintf("%.1f", *card.CertifiedAvgScore)
	}
	w("## Summary\n\n")
	w("%d members · %d certified (current) · %d stale · %d report-missing · %d defined · %d planned · certified avg %s\n\n",
		card.MemberCount, card.Counts.CertifiedCurrent, card.Counts.CertifiedStale, card.Counts.ReportMissing,
		card.Counts.Defined, card.Counts.Planned, avg)
	w("## Members\n\n")
	w("| Agent | Declared | State | Score | Verdict | Note |\n|---|---|---|---:|---|---|\n")
	for _, m := range card.Members {
		score := ""
		if m.Score != nil {
			score = fmt.Sprintf("%.1f", *m.Score)
		}
		w("| %s | %s | %s | %s | %s | %s |\n", m.AgentID, m.DeclaredStatus, m.State, score, m.Verdict, m.Note)
	}
	w("\nCertified scores are read from each member's committed readiness report; a member declared certified without a report is shown as report-missing, not counted as certified.\n")
	return sb.String()
}
