package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage StinkyKitty configuration",
	Long:  "View and modify StinkyKitty configuration values",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		value := config.GetString(args[0])
		fmt.Println(value)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := config.Set(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Set %s = %s\n", args[0], args[1])
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		all := config.GetAll()
		for key, value := range all {
			fmt.Printf("%s: %v\n", key, value)
		}
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

// initConfig initializes the configuration system
func initConfig() error {
	configPath := os.Getenv("STINKY_CONFIG")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = home + "/.stinkykitty/config.yaml"
	}

	return config.InitConfig(configPath)
}
