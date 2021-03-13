package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "v0.0.3"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "displays the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gomental version : %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
