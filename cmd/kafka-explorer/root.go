package kafka_explorer

import (
	"fmt"
	"os"

	"github.com/kafka-explorer-cli/internal/core/ports"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile          string
	brokers          []string
	topic            string
	group            string
	searchStr        string
	limit            int
	caseSensitive    bool
	useRegex         bool
	outputFile       string
	verbose          bool
	saslEnabled      bool
	saslMechanism    string
	securityProtocol string
	username         string
	password         string
)

var rootCmd = &cobra.Command{
	Use:   "kafka-explorer",
	Short: "A CLI tool for exploring Kafka topics",
	Long: `Kafka Explorer is a CLI utility for searching and reading messages from Kafka topics.
It allows for real-time search and display of results with high performance.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kafka-explorer.yaml)")
}

// getSASLConfig creates a SASLConfig from command line flags
func getSASLConfig() ports.SASLConfig {
	return ports.SASLConfig{
		Enabled:   saslEnabled,
		Mechanism: saslMechanism,
		Username:  username,
		Password:  password,
		Protocol:  securityProtocol,
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kafka-explorer" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kafka-explorer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}