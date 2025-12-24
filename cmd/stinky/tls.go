package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/tls"
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "TLS certificate management",
	Long:  "Manage SSL/TLS certificates for StinkyKitty sites",
}

var tlsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show certificate status",
	Long:  "Display the status of all managed SSL/TLS certificates",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize config
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Check if TLS is enabled
		if !config.GetBool("server.tls_enabled") {
			fmt.Println("TLS is disabled. Enable it with: stinky config set server.tls_enabled true")
			os.Exit(0)
		}

		// Initialize database
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Load TLS config
		tlsCfg, err := tls.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load TLS config: %v\n", err)
			os.Exit(1)
		}

		// Create TLS manager
		tlsManager, err := tls.NewManager(db.GetDB(), tlsCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create TLS manager: %v\n", err)
			os.Exit(1)
		}

		// Get certificate status
		statuses, err := tlsManager.GetCertificateStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get certificate status: %v\n", err)
			os.Exit(1)
		}

		if len(statuses) == 0 {
			fmt.Println("No certificates found. Certificates are provisioned on first HTTPS request to a domain.")
			fmt.Println("\nConfigured domains:")
			domains, _ := tlsManager.GetAllowedDomains()
			for _, domain := range domains {
				fmt.Printf("  - %s (not yet provisioned)\n", domain)
			}
			os.Exit(0)
		}

		// Display certificate status
		fmt.Printf("%-30s %-20s %-15s %s\n", "Domain", "Issuer", "Expires", "Days Left")
		fmt.Println("-----------------------------------------------------------------------------------")
		for _, status := range statuses {
			fmt.Printf("%-30s %-20s %-15s %d\n",
				status.Domain,
				status.Issuer,
				status.NotAfter.Format("2006-01-02"),
				status.DaysUntilExpiry,
			)
		}
	},
}

func init() {
	tlsCmd.AddCommand(tlsStatusCmd)
	rootCmd.AddCommand(tlsCmd)
}
