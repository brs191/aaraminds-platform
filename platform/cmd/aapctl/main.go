package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	aapruntime "github.com/aaraminds/aaraminds-platform/platform/internal/runtime"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "contracts":
		listContracts(os.Args[2:])
	case "mcp-tools":
		mcpTools(os.Args[2:])
	case "prove":
		prove(os.Args[2:])
	case "validate":
		validate(os.Args[2:])
	case "intake":
		intake(os.Args[2:])
	case "classify":
		classify(os.Args[2:])
	case "scaffold":
		scaffold(os.Args[2:])
	case "sections":
		sections(os.Args[2:])
	case "readiness":
		readiness(os.Args[2:])
	case "export":
		export(os.Args[2:])
	case "pack":
		pack(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func intake(args []string) {
	fs := flag.NewFlagSet("intake", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	path := fs.String("file", "examples/ba-agent.intake.yaml", "intake file path relative to root")
	_ = fs.Parse(args)

	record, err := aapruntime.LoadIntake(*root, filepath.Join(*root, *path))
	if err != nil {
		fmt.Fprintln(os.Stderr, "intake validation failed:", err)
		os.Exit(1)
	}
	fmt.Printf("intake valid: %s (submitted by %s)\n", record.AgentID, record.SubmittedBy)
	fmt.Printf("  execution_intent: %s | tools: %d | data domains: %d\n",
		record.ExecutionIntent, len(record.ProposedTools), len(record.DataDomains))
}

func classify(args []string) {
	fs := flag.NewFlagSet("classify", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	path := fs.String("file", "examples/ba-agent.intake.yaml", "intake file path relative to root")
	_ = fs.Parse(args)

	record, err := aapruntime.LoadIntake(*root, filepath.Join(*root, *path))
	if err != nil {
		fmt.Fprintln(os.Stderr, "intake validation failed:", err)
		os.Exit(1)
	}
	classification, err := aapruntime.ClassifyAgent(record)
	if err != nil {
		fmt.Fprintln(os.Stderr, "classification failed:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(classification); err != nil {
		fmt.Fprintln(os.Stderr, "encode classification:", err)
		os.Exit(1)
	}
}

func listContracts(args []string) {
	fs := flag.NewFlagSet("contracts", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	contractsDir := fs.String("contracts", "tool-contracts", "tool contracts directory relative to root")
	_ = fs.Parse(args)

	contracts, err := aapruntime.LoadContractsWithSchema(filepath.Join(*root, *contractsDir), filepath.Join(*root, "schemas", "mcp-tool-contract.schema.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "load contracts:", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TOOL\tVERSION\tACTION\tBOUNDARY\tTIMEOUT_MS\tRETRY")
	for _, name := range names {
		contract := contracts[name]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s/%d\n",
			contract.ToolName,
			contract.ContractVersion,
			contract.ActionType,
			contract.ApprovalBoundary,
			contract.TimeoutMS,
			contract.RetryPolicy.Backoff,
			contract.RetryPolicy.MaxAttempts,
		)
	}
	_ = w.Flush()
}

func mcpTools(args []string) {
	fs := flag.NewFlagSet("mcp-tools", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	contractsDir := fs.String("contracts", "tool-contracts", "tool contracts directory relative to root")
	_ = fs.Parse(args)

	contracts, err := aapruntime.LoadContractsWithSchema(filepath.Join(*root, *contractsDir), filepath.Join(*root, "schemas", "mcp-tool-contract.schema.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "load contracts:", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	sort.Strings(names)

	type mcpTool struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		InputSchema map[string]any `json:"inputSchema"`
		Annotations map[string]any `json:"annotations"`
	}
	out := struct {
		Tools []mcpTool `json:"tools"`
	}{Tools: make([]mcpTool, 0, len(names))}
	for _, name := range names {
		contract := contracts[name]
		out.Tools = append(out.Tools, mcpTool{
			Name:        contract.ToolName,
			Description: contract.Purpose,
			InputSchema: contract.InputSchema,
			Annotations: map[string]any{
				"aap.action_type":       contract.ActionType,
				"aap.contract_version":  contract.ContractVersion,
				"aap.approval_boundary": contract.ApprovalBoundary,
				"aap.timeout_ms":        contract.TimeoutMS,
				"aap.retry_backoff":     contract.RetryPolicy.Backoff,
				"aap.max_retries":       contract.RetryPolicy.MaxAttempts,
			},
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode mcp tools:", err)
		os.Exit(1)
	}
}

func prove(args []string) {
	fs := flag.NewFlagSet("prove", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	out := fs.String("out", "out/proofs/phase1-proof.json", "proof report path relative to root")
	otelCfg := aapruntime.OTelConfigFromEnv("aaraminds-aap-runtime", "1.0.0")
	otelEnabled := fs.Bool("otel", otelCfg.Enabled, "emit OpenTelemetry spans for the proof run")
	otelExporter := fs.String("otel-exporter", otelCfg.Exporter, "OpenTelemetry trace exporter: stdout, otlp, or none")
	otelEndpoint := fs.String("otel-endpoint", otelCfg.Endpoint, "OTLP gRPC endpoint, for example localhost:4317")
	otelInsecure := fs.Bool("otel-insecure", otelCfg.Insecure, "use insecure OTLP gRPC transport")
	_ = fs.Parse(args)

	otelCfg.Enabled = *otelEnabled
	otelCfg.Exporter = *otelExporter
	otelCfg.Endpoint = *otelEndpoint
	otelCfg.Insecure = *otelInsecure
	if !flagSet(fs, "otel-insecure") &&
		os.Getenv("AAP_OTEL_INSECURE") == "" &&
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(otelCfg.Endpoint)), "https://") {
		otelCfg.Insecure = false
	}
	shutdown, err := aapruntime.ConfigureOpenTelemetry(context.Background(), otelCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configure otel:", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "shutdown otel:", err)
		}
	}()

	report, err := aapruntime.RunPhase1Proof(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "proof failed:", err)
		os.Exit(1)
	}
	if err := aapruntime.WriteProofReport(*root, report, *out); err != nil {
		fmt.Fprintln(os.Stderr, "write proof:", err)
		os.Exit(1)
	}
	fmt.Printf("phase1 proof written: %s\n", filepath.Join(*root, *out))
}

func validate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	manifest := fs.String("manifest", "examples/ba-agent.manifest.yaml", "manifest path relative to root")
	contracts := fs.String("contracts", "tool-contracts", "tool contracts directory relative to root")
	_ = fs.Parse(args)

	if _, err := aapruntime.NewEngine(*root, *manifest, *contracts); err != nil {
		fmt.Fprintln(os.Stderr, "validation failed:", err)
		os.Exit(1)
	}
	fmt.Println("manifest and tool contracts validated")
}

func scaffold(args []string) {
	fs := flag.NewFlagSet("scaffold", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	path := fs.String("file", "examples/ba-agent.intake.yaml", "intake file path relative to root")
	out := fs.String("out", "agents", "output directory relative to root")
	force := fs.Bool("force", false, "overwrite a non-empty target directory")
	_ = fs.Parse(args)

	dir, files, err := aapruntime.ScaffoldAgent(*root, filepath.Join(*root, *path), filepath.Join(*root, *out), *force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scaffold failed:", err)
		os.Exit(1)
	}
	fmt.Printf("scaffolded %d artifacts in %s\n", len(files), dir)
	for _, f := range files {
		fmt.Println("  ", filepath.Base(f))
	}
	fmt.Println("section self-check: all artifacts passed")
}

func sections(args []string) {
	fs := flag.NewFlagSet("sections", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	dir := fs.String("dir", "", "agent artifact directory relative to root (absolute paths used as-is)")
	_ = fs.Parse(args)
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "sections: -dir is required")
		os.Exit(2)
	}
	target := *dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(*root, target)
	}
	reports, err := aapruntime.ValidateArtifactDir(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "section validation error:", err)
		os.Exit(1)
	}
	failed := false
	for _, report := range reports {
		if report.OK() {
			fmt.Printf("ok    %s\n", filepath.Base(report.Artifact))
			continue
		}
		failed = true
		fmt.Printf("FAIL  %s missing=%v empty=%v\n", filepath.Base(report.Artifact), report.Missing, report.Empty)
	}
	if failed {
		os.Exit(1)
	}
}

// readiness runs the rubric against an agent artifact directory, writes
// readiness-report.json + .md, and exits: 0 pass, 3 defer, 4 block, 1 error.
// The exit codes let CI treat defer and block differently.
func readiness(args []string) {
	fs := flag.NewFlagSet("readiness", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	agent := fs.String("agent", "agents/aara-business-analyst", "agent artifact directory relative to root")
	manifest := fs.String("manifest", "examples/ba-agent.manifest.yaml", "manifest path relative to root (empty to skip manifest checks)")
	_ = fs.Parse(args)

	agentDir := filepath.Join(*root, *agent)
	manifestPath := ""
	if *manifest != "" {
		manifestPath = filepath.Join(*root, *manifest)
	}
	report, err := aapruntime.RunReadiness(*root, agentDir, manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "readiness failed:", err)
		os.Exit(1)
	}
	if err := aapruntime.WriteReadinessReport(*root, agentDir, report); err != nil {
		fmt.Fprintln(os.Stderr, "write readiness report:", err)
		os.Exit(1)
	}
	fmt.Printf("readiness: %s scored %.1f/100 -> %s (rubric %s)\n", report.AgentID, report.Score, strings.ToUpper(report.Verdict), report.RubricVersion)
	for _, blocker := range report.CriticalBlockers {
		fmt.Printf("  CRITICAL %s: %s\n", blocker.BlockerID, blocker.RequiredFix)
	}
	failing := 0
	for _, area := range report.Areas {
		failing += area.ChecksTotal - area.ChecksPassed
	}
	fmt.Printf("  checks failing: %d; report: %s\n", failing, filepath.Join(agentDir, "readiness-report.md"))

	// Activation gate advisory: if the paired manifest claims active status,
	// enforce the verdict now.
	if manifestPath != "" {
		if m, err := aapruntime.LoadManifest(manifestPath); err == nil {
			if err := aapruntime.ActivationGate(*root, agentDir, m.Status); err != nil {
				fmt.Fprintln(os.Stderr, "ACTIVATION GATE:", err)
				os.Exit(4)
			}
		}
	}
	switch report.Verdict {
	case "pass":
		os.Exit(0)
	case "defer":
		os.Exit(3)
	default:
		os.Exit(4)
	}
}

// export copies an agent folder with a tamper-evident hash manifest and,
// with -verify, proves the AC-009 round trip and writes the attestation.
func export(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	agent := fs.String("agent", "agents/aara-business-analyst", "agent artifact directory relative to root")
	dest := fs.String("dest", "", "export destination relative to root (default out/exports/<agent-id>)")
	manifest := fs.String("manifest", "examples/ba-agent.manifest.yaml", "manifest path relative to root (used by -verify)")
	verify := fs.Bool("verify", false, "run the export/re-import round trip and write export-verification.json")
	_ = fs.Parse(args)

	agentDir := filepath.Join(*root, *agent)

	if *verify {
		scratch, err := os.MkdirTemp("", "aap-roundtrip-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, "scratch dir:", err)
			os.Exit(1)
		}
		defer os.RemoveAll(scratch)
		manifestPath := ""
		if *manifest != "" {
			manifestPath = filepath.Join(*root, *manifest)
		}
		verification, err := aapruntime.RoundTripVerify(*root, agentDir, manifestPath, scratch)
		if err != nil {
			fmt.Fprintln(os.Stderr, "round-trip verification failed:", err)
			os.Exit(1)
		}
		fmt.Printf("round-trip verified: %s — %d checks identical (score %.1f, %s); attestation written\n",
			verification.AgentID, verification.ChecksCompared, verification.ReportScore, verification.ReportVerdict)
		return
	}

	destDir := *dest
	if destDir == "" {
		destDir = filepath.Join("out", "exports", filepath.Base(agentDir))
	}
	exportManifest, err := aapruntime.ExportAgent(*root, agentDir, filepath.Join(*root, destDir))
	if err != nil {
		fmt.Fprintln(os.Stderr, "export failed:", err)
		os.Exit(1)
	}
	if err := aapruntime.VerifyExport(*root, filepath.Join(*root, destDir)); err != nil {
		fmt.Fprintln(os.Stderr, "export self-verification failed:", err)
		os.Exit(1)
	}
	fmt.Printf("exported %s: %d files to %s (integrity verified)\n", exportManifest.AgentID, len(exportManifest.Files), destDir)
}

// pack rolls up member readiness for one pack (-pack) or all packs (-all),
// writing scorecards to out/packs/ and printing a summary line per pack.
func pack(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	packPath := fs.String("pack", "", "pack manifest path relative to root")
	all := fs.Bool("all", false, "roll up every pack in packs/")
	out := fs.String("out", "out/packs", "scorecard output directory relative to root")
	_ = fs.Parse(args)

	var packs []aapruntime.Pack
	var err error
	switch {
	case *all:
		packs, err = aapruntime.LoadAllPacks(*root)
	case *packPath != "":
		var p aapruntime.Pack
		p, err = aapruntime.LoadPack(*root, filepath.Join(*root, *packPath))
		packs = []aapruntime.Pack{p}
	default:
		fmt.Fprintln(os.Stderr, "pack: provide -pack <path> or -all")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "load packs:", err)
		os.Exit(1)
	}
	outDir := filepath.Join(*root, *out)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create out dir:", err)
		os.Exit(1)
	}
	issues := 0
	for _, p := range packs {
		card, err := aapruntime.RunPackReadiness(*root, p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pack readiness:", err)
			os.Exit(1)
		}
		if err := aapruntime.WritePackScorecard(*root, outDir, card); err != nil {
			fmt.Fprintln(os.Stderr, "write scorecard:", err)
			os.Exit(1)
		}
		avg := "n/a"
		if card.CertifiedAvgScore != nil {
			avg = fmt.Sprintf("%.1f", *card.CertifiedAvgScore)
		}
		fmt.Printf("%-24s [%s] %d members: %d certified, %d defined, %d planned; certified avg %s\n",
			card.PackID, card.Timeline, card.MemberCount,
			card.Counts.CertifiedCurrent, card.Counts.Defined, card.Counts.Planned, avg)
		if card.Counts.ReportMissing > 0 || card.Counts.CertifiedStale > 0 {
			issues++
			fmt.Printf("  ! %d report-missing, %d stale — a member declared certified lacks a current report\n",
				card.Counts.ReportMissing, card.Counts.CertifiedStale)
		}
	}
	if issues > 0 {
		os.Exit(1)
	}
}

func defaultRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".."
	}
	if filepath.Base(wd) == "platform" {
		return filepath.Clean(filepath.Join(wd, ".."))
	}
	return wd
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: aapctl <contracts|mcp-tools|prove|validate|intake|classify|scaffold|sections|readiness|export|pack> [flags]")
	fmt.Fprintln(os.Stderr, "readiness exit codes: 0 pass, 3 defer, 4 block (or activation gate violation), 1 error")
}

func flagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
