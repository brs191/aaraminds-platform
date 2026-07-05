package aapruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Epic 9 (AC-009): deterministic export with a tamper-evident hash manifest,
// verification of exported copies, and a round-trip attestation proving that
// re-importing the folder reproduces identical readiness results.
//
// Design note: the readiness check "export-roundtrip" cannot run the round
// trip itself — a readiness run inside a readiness run would recurse. Instead
// RoundTripVerify writes export-verification.json bound to a digest of the
// artifact bytes it verified; the readiness check recomputes the digest and
// fails if artifacts changed since verification. Stale attestations fail
// closed.

// ExportManifest mirrors schemas/export-manifest.schema.json.
type ExportManifest struct {
	AgentID    string       `json:"agent_id"`
	ExportedAt string       `json:"exported_at"`
	Files      []ExportFile `json:"files"`
}

type ExportFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// ExportVerification mirrors schemas/export-verification.schema.json.
type ExportVerification struct {
	AgentID        string  `json:"agent_id"`
	VerifiedAt     string  `json:"verified_at"`
	ContentDigest  string  `json:"content_digest"`
	Identical      bool    `json:"identical"`
	ReportScore    float64 `json:"report_score"`
	ReportVerdict  string  `json:"report_verdict"`
	ChecksCompared int     `json:"checks_compared"`
}

const (
	exportManifestName     = "export-manifest.json"
	exportVerificationName = "export-verification.json"
)

// exportableFiles returns the relative paths in an agent directory that an
// export carries: every scaffold-owned artifact plus optional governance
// records (sign-offs, eval runs) and derived reports if present. Sorted for
// determinism.
func exportableFiles(agentDir string) ([]string, error) {
	var files []string
	add := func(rel string) {
		if _, err := os.Stat(filepath.Join(agentDir, rel)); err == nil {
			files = append(files, rel)
		}
	}
	for _, name := range sortedArtifactNames() {
		add(name)
	}
	for _, name := range generatedJSONArtifacts {
		add(name)
	}
	for _, name := range generatedExtras {
		add(name)
	}
	add("signoffs.json")
	add("readiness-report.json")
	add("readiness-report.md")
	// The verification record travels with the export so a readiness run on
	// the imported copy sees identical state to the source.
	add(exportVerificationName)
	if entries, err := os.ReadDir(filepath.Join(agentDir, "eval-runs")); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				files = append(files, filepath.Join("eval-runs", entry.Name()))
			}
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("nothing exportable in %s", agentDir)
	}
	return files, nil
}

// contentDigestFiles are the round-trip attestation inputs: everything the
// readiness engine reads, excluding derived outputs (readiness reports, the
// attestation itself, the export manifest) whose bytes change per run.
func contentDigestFiles(agentDir string) ([]string, error) {
	all, err := exportableFiles(agentDir)
	if err != nil {
		return nil, err
	}
	var inputs []string
	for _, rel := range all {
		switch rel {
		case "readiness-report.json", "readiness-report.md", exportManifestName, exportVerificationName:
			continue
		}
		inputs = append(inputs, rel)
	}
	return inputs, nil
}

// ContentDigest computes a single sha256 over the sorted (path, filehash)
// pairs of the readiness input files. Any added, removed, or modified input
// changes the digest.
func ContentDigest(agentDir string) (string, error) {
	inputs, err := contentDigestFiles(agentDir)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, rel := range inputs {
		fileHash, _, err := hashFile(filepath.Join(agentDir, rel))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%s\x00", filepath.ToSlash(rel), fileHash)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ExportAgent copies the agent folder to destDir and writes a hash manifest.
// destDir must not already contain an export.
func ExportAgent(root, agentDir, destDir string) (ExportManifest, error) {
	var manifest ExportManifest
	files, err := exportableFiles(agentDir)
	if err != nil {
		return manifest, err
	}
	if _, err := os.Stat(filepath.Join(destDir, exportManifestName)); err == nil {
		return manifest, fmt.Errorf("destination %s already contains an export", destDir)
	}
	var intake AgentIntake
	if _, err := loadStructuredFile(filepath.Join(agentDir, "agent-intake.yaml"), &intake); err != nil {
		return manifest, fmt.Errorf("export requires agent-intake.yaml: %w", err)
	}
	manifest.AgentID = intake.AgentID
	manifest.ExportedAt = time.Now().UTC().Format(time.RFC3339)

	for _, rel := range files {
		src := filepath.Join(agentDir, rel)
		fileHash, size, err := hashFile(src)
		if err != nil {
			return manifest, err
		}
		dst := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return manifest, fmt.Errorf("export mkdir: %w", err)
		}
		raw, err := os.ReadFile(src)
		if err != nil {
			return manifest, fmt.Errorf("export read %s: %w", src, err)
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return manifest, fmt.Errorf("export write %s: %w", dst, err)
		}
		manifest.Files = append(manifest.Files, ExportFile{Path: filepath.ToSlash(rel), SHA256: fileHash, Bytes: size})
	}
	manifestPath := filepath.Join(destDir, exportManifestName)
	if err := writeJSONArtifact(manifestPath, manifest); err != nil {
		return manifest, err
	}
	if err := ValidateStructuredFile(manifestPath, filepath.Join(root, "schemas", "export-manifest.schema.json")); err != nil {
		return manifest, fmt.Errorf("generated export manifest failed its own schema: %w", err)
	}
	return manifest, nil
}

// VerifyExport checks an exported folder against its manifest: every listed
// file must exist with a matching hash, and no unlisted files may be present.
// Both directions matter — a dropped file and a smuggled file are equally
// integrity failures.
func VerifyExport(root, exportDir string) error {
	manifestPath := filepath.Join(exportDir, exportManifestName)
	if err := ValidateStructuredFile(manifestPath, filepath.Join(root, "schemas", "export-manifest.schema.json")); err != nil {
		return fmt.Errorf("export manifest invalid: %w", err)
	}
	var manifest ExportManifest
	if _, err := loadStructuredFile(manifestPath, &manifest); err != nil {
		return fmt.Errorf("load export manifest: %w", err)
	}
	listed := map[string]bool{exportManifestName: true}
	for _, file := range manifest.Files {
		listed[file.Path] = true
		path := filepath.Join(exportDir, filepath.FromSlash(file.Path))
		fileHash, size, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("export integrity: listed file %s: %w", file.Path, err)
		}
		if fileHash != file.SHA256 || size != file.Bytes {
			return fmt.Errorf("export integrity: %s does not match its manifest hash", file.Path)
		}
	}
	return filepath.WalkDir(exportDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(exportDir, path)
		if err != nil {
			return err
		}
		if !listed[filepath.ToSlash(rel)] {
			return fmt.Errorf("export integrity: unlisted file %s present in export", rel)
		}
		return nil
	})
}

// RoundTripVerify proves AC-009: export the agent folder, verify integrity,
// run readiness on both source and imported copy, and require identical
// normalized results. On success it writes export-verification.json into the
// agent directory, bound to the current content digest.
func RoundTripVerify(root, agentDir, manifestPath, scratchDir string) (ExportVerification, error) {
	var verification ExportVerification

	exportDir := filepath.Join(scratchDir, "export")
	if _, err := ExportAgent(root, agentDir, exportDir); err != nil {
		return verification, fmt.Errorf("round-trip export: %w", err)
	}
	if err := VerifyExport(root, exportDir); err != nil {
		return verification, fmt.Errorf("round-trip integrity: %w", err)
	}

	source, err := RunReadiness(root, agentDir, manifestPath)
	if err != nil {
		return verification, fmt.Errorf("round-trip source readiness: %w", err)
	}
	imported, err := RunReadiness(root, exportDir, manifestPath)
	if err != nil {
		return verification, fmt.Errorf("round-trip imported readiness: %w", err)
	}

	checks, diff := compareReadiness(source, imported)
	if diff != "" {
		return verification, fmt.Errorf("round-trip mismatch: %s", diff)
	}

	digest, err := ContentDigest(agentDir)
	if err != nil {
		return verification, err
	}
	verification = ExportVerification{
		AgentID:        source.AgentID,
		VerifiedAt:     time.Now().UTC().Format(time.RFC3339),
		ContentDigest:  digest,
		Identical:      true,
		ReportScore:    source.Score,
		ReportVerdict:  source.Verdict,
		ChecksCompared: checks,
	}
	verificationPath := filepath.Join(agentDir, exportVerificationName)
	if err := writeJSONArtifact(verificationPath, verification); err != nil {
		return verification, err
	}
	if err := ValidateStructuredFile(verificationPath, filepath.Join(root, "schemas", "export-verification.schema.json")); err != nil {
		return verification, fmt.Errorf("generated verification failed its own schema: %w", err)
	}
	return verification, nil
}

// compareReadiness compares two reports modulo timestamps and evidence path
// prefixes (the imported copy lives in a different directory). Identical
// means: same score, same verdict, same blockers, and the same result for
// every check id.
func compareReadiness(a, b ReadinessReport) (int, string) {
	if a.Score != b.Score {
		return 0, fmt.Sprintf("score %.2f != %.2f", a.Score, b.Score)
	}
	if a.Verdict != b.Verdict {
		return 0, fmt.Sprintf("verdict %s != %s", a.Verdict, b.Verdict)
	}
	if len(a.CriticalBlockers) != len(b.CriticalBlockers) {
		return 0, fmt.Sprintf("critical blockers %d != %d", len(a.CriticalBlockers), len(b.CriticalBlockers))
	}
	resultsA := checkResults(a)
	resultsB := checkResults(b)
	if len(resultsA) != len(resultsB) {
		return 0, fmt.Sprintf("check count %d != %d", len(resultsA), len(resultsB))
	}
	for id, result := range resultsA {
		other, ok := resultsB[id]
		if !ok {
			return 0, fmt.Sprintf("check %s missing from imported report", id)
		}
		if result != other {
			return 0, fmt.Sprintf("check %s: %s != %s", id, result, other)
		}
	}
	return len(resultsA), ""
}

func checkResults(report ReadinessReport) map[string]string {
	results := map[string]string{}
	for _, area := range report.Areas {
		for _, check := range area.Evidence {
			results[check.Check] = check.Result
		}
	}
	return results
}

func hashFile(path string) (string, int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), int64(len(raw)), nil
}
