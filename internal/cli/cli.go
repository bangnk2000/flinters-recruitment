// Package cli parses the command line, validates the input and output paths,
// and maps every failure to a process exit code.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// Exit codes returned by Run.
const (
	ExitOK    = 0 // success, or --help
	ExitFault = 1 // I/O or runtime failure
	ExitUsage = 2 // invalid command line
)

const usageText = `Usage: aggregator --input <file.csv> --output <dir>

Aggregates ad performance per campaign_id and writes top-10 CTR / CPA reports.

Flags:
  --input   path to the input CSV file (required)
  --output  directory for the result files, created if missing (required)
`

type config struct {
	input  string
	output string
}

// Run executes the CLI with args (without the program name) and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseArgs(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, usageText)
		return ExitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "aggregator: %v\n\n%s", err, usageText)
		return ExitUsage
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(stderr, "aggregator: %v\n", err)
		return ExitFault
	}
	return ExitOK
}

func parseArgs(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("aggregator", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.input, "input", "", "")
	fs.StringVar(&cfg.output, "output", "", "")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	switch {
	case fs.NArg() > 0:
		return config{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	case cfg.input == "":
		return config{}, errors.New("--input is required")
	case cfg.output == "":
		return config{}, errors.New("--output is required")
	}
	return cfg, nil
}

func run(cfg config) error {
	in, err := os.Open(cfg.input)
	if err != nil {
		return fmt.Errorf("cannot read input: %w", err)
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("cannot read input: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("cannot read input: %s is a directory", cfg.input)
	}

	if err := os.MkdirAll(cfg.output, 0o755); err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}
	return nil
}
