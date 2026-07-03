package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	aapruntime "github.com/aaraminds/aaraminds-platform/platform/internal/runtime"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "prove":
		prove(os.Args[2:])
	case "validate":
		validate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func prove(args []string) {
	fs := flag.NewFlagSet("prove", flag.ExitOnError)
	root := fs.String("root", defaultRoot(), "repository root")
	out := fs.String("out", "out/proofs/phase1-proof.json", "proof report path relative to root")
	_ = fs.Parse(args)

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
	fmt.Fprintln(os.Stderr, "usage: aapctl <prove|validate> [flags]")
}
