package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sphinxid/bacot/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print bacot version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bacot v%s (commit: %s, built: %s)\n",
			version.Version, version.Commit, version.BuildDate)
	},
}
