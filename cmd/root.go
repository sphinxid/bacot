// Package cmd contains all CLI commands for bacot.
package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// outputFlag holds the --output flag value (e.g. "json=result.json").
var outputFlag []string

// noColorFlag disables terminal colors when true.
var noColorFlag bool

// quietFlag suppresses the live dashboard when true.
var quietFlag bool

// fullReportFlag enables the full per-scenario metrics report after the summary.
var fullReportFlag bool

// rootCmd is the base command for the bacot CLI.
var rootCmd = &cobra.Command{
	Use:   "bacot",
	Short: "bacot — a production-grade HTTP performance testing tool",
	Long: `bacot is a fast, scriptable HTTP load testing CLI tool.
Run a test from a YAML script:  bacot run script.yaml
Run a quick inline test:         bacot run --url https://example.com --vus 10 --duration 30s`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringArrayVar(&outputFlag, "output", nil, "output format and destination, e.g. --output json=result.json or --output html=report.html")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "disable terminal color output")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "suppress live dashboard; only show final summary")
	rootCmd.PersistentFlags().BoolVar(&fullReportFlag, "full-report", false, "print full per-scenario metrics report after the summary")

	cobra.OnInitialize(func() {
		if noColorFlag {
			color.NoColor = true
		}
	})

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

// parseOutputFlag parses the output flag entries and returns (format, path) pairs.
// Supported formats: "json=<path>", "html=<path>".
func parseOutputFlag(flags []string) map[string]string {
	result := make(map[string]string)
	for _, f := range flags {
		for _, prefix := range []string{"json=", "html="} {
			if len(f) > len(prefix) && f[:len(prefix)] == prefix {
				result[f[:len(prefix)-1]] = f[len(prefix):]
			}
		}
	}
	return result
}
