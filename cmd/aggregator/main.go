// Command aggregator streams an ad-performance CSV and writes per-campaign top-10 reports.
package main

import (
	"os"

	"adagg/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
