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
	fmt.Fprintln(os.Stderr, "usage: aapctl <contracts|mcp-tools|prove|validate|intake|classify> [flags]")
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
