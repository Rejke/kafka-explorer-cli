package kafka_explorer

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// Version command implementation
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version, build date, and git commit of Kafka Explorer CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kafka Explorer CLI\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}