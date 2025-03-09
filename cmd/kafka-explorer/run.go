package kafka_explorer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kafka-explorer-cli/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Run command implementation
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Kafka topic explorer",
	Long:  `Run the Kafka topic explorer to search and display messages from Kafka topics`,
	RunE:  runExplorer,
	// Inherit all flags from parent
	TraverseChildren: true,
}

// runExplorer runs the Kafka explorer application
func runExplorer(cmd *cobra.Command, args []string) error {
	// Create a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
	}()

	// Create application configuration
	config := app.Config{
		Brokers: brokers,
		Topic:   topic,
		Group:   group,

		SearchString:  searchStr,
		CaseSensitive: caseSensitive,
		UseRegex:      useRegex,
		ContextSize:   100, // Number of characters to include before and after the match

		OutputFile: outputFile,
		Limit:      limit,

		SASL: getSASLConfig(),
	}

	// Create and run the explorer
	explorer := app.NewExplorer(config)
	if err := explorer.Run(ctx); err != nil {
		return fmt.Errorf("explorer failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Kafka connection parameters
	runCmd.Flags().StringSliceVar(&brokers, "brokers", []string{"localhost:9092"}, "Kafka broker addresses")
	runCmd.Flags().StringVar(&topic, "topic", "", "Kafka topic to read from")
	runCmd.Flags().StringVar(&group, "group", "kafka-explorer", "Consumer group ID")

	// SASL Authentication parameters
	runCmd.Flags().BoolVar(&saslEnabled, "sasl", false, "Enable SASL authentication")
	runCmd.Flags().StringVar(&saslMechanism, "sasl-mechanism", "PLAIN", "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
	runCmd.Flags().StringVar(&securityProtocol, "security-protocol", "SASL_PLAINTEXT", "Security protocol (PLAINTEXT, SASL_PLAINTEXT, SSL, SASL_SSL)")
	runCmd.Flags().StringVar(&username, "username", "", "SASL username")
	runCmd.Flags().StringVar(&password, "password", "", "SASL password")

	// Search parameters
	runCmd.Flags().StringVar(&searchStr, "search", "", "String to search for in messages")
	runCmd.Flags().IntVar(&limit, "limit", 0, "Number of matches to find before stopping (0 = unlimited)")
	runCmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Enable case-sensitive search")
	runCmd.Flags().BoolVar(&useRegex, "regex", false, "Use regular expressions for search")

	// Output parameters
	runCmd.Flags().StringVar(&outputFile, "output", "results.json", "Path to output JSON file")
	runCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	// Required flags
	if err := runCmd.MarkFlagRequired("topic"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag 'topic' as required: %v\n", err)
	}
	if err := runCmd.MarkFlagRequired("search"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag 'search' as required: %v\n", err)
	}

	// Mark SASL authentication flags as required when SASL is enabled
	runCmd.MarkFlagsRequiredTogether("sasl", "username", "password")

	// Bind flags to viper
	if err := viper.BindPFlags(runCmd.Flags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags to viper: %v\n", err)
	}
}
