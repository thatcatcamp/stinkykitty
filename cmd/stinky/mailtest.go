// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/email"
)

var mailtestCmd = &cobra.Command{
	Use:   "mailtest <email>",
	Short: "Test email configuration and send a test message",
	Long: `Test the email service by sending a test message to the specified address.
This command checks SMTP configuration and provides detailed diagnostics.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		testEmail := args[0]

		fmt.Println("========================================")
		fmt.Println("StinkyKitty Email Service Diagnostic")
		fmt.Println("========================================")
		fmt.Println()

		// Check environment variables
		fmt.Println("1. Checking environment variables...")
		smtp := os.Getenv("SMTP")
		smtpPort := os.Getenv("SMTP_PORT")
		emailFrom := os.Getenv("EMAIL")
		smtpSecret := os.Getenv("SMTP_SECRET")

		fmt.Printf("   SMTP:        %s\n", maskIfEmpty(smtp))
		fmt.Printf("   SMTP_PORT:   %s\n", maskIfEmpty(smtpPort))
		fmt.Printf("   EMAIL:       %s\n", maskIfEmpty(emailFrom))
		fmt.Printf("   SMTP_SECRET: %s\n", maskPassword(smtpSecret))
		fmt.Println()

		if smtp == "" || smtpPort == "" || emailFrom == "" || smtpSecret == "" {
			fmt.Println("❌ ERROR: Missing required environment variables")
			fmt.Println()
			fmt.Println("Required environment variables:")
			fmt.Println("  export SMTP=\"smtp.example.com\"")
			fmt.Println("  export SMTP_PORT=\"587\"")
			fmt.Println("  export EMAIL=\"noreply@yourdomain.com\"")
			fmt.Println("  export SMTP_SECRET=\"your-password\"")
			fmt.Println()
			fmt.Println("See docs/EMAIL_SETUP.md for detailed configuration instructions.")
			os.Exit(1)
		}

		fmt.Println("✓ Environment variables are set")
		fmt.Println()

		// Check port configuration
		fmt.Println("2. Checking port configuration...")
		if smtpPort == "587" {
			fmt.Println("   Using port 587 (STARTTLS)")
		} else if smtpPort == "465" {
			fmt.Println("   Using port 465 (Implicit TLS)")
		} else {
			fmt.Printf("   ⚠️  WARNING: Unusual port %s (standard ports are 587 or 465)\n", smtpPort)
		}
		fmt.Println()

		// Initialize email service
		fmt.Println("3. Initializing email service...")
		svc, err := email.NewEmailService()
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to initialize email service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Email service initialized")
		fmt.Println()

		// Send test email
		fmt.Printf("4. Sending test email to %s...\n", testEmail)
		subject := "StinkyKitty Email Test"
		body := fmt.Sprintf(`Hello,

This is a test email from StinkyKitty CMS.

If you received this email, your email configuration is working correctly!

Test Details:
- Sent at: %s
- From: %s
- SMTP Server: %s:%s

Best regards,
StinkyKitty Email Service`, time.Now().Format("2006-01-02 15:04:05 MST"), emailFrom, smtp, smtpPort)

		if err := svc.SendEmail(testEmail, subject, body); err != nil {
			fmt.Println("❌ ERROR: Failed to send test email")
			fmt.Println()
			fmt.Println("Error details:")
			fmt.Printf("  %v\n", err)
			fmt.Println()
			fmt.Println("Common issues and solutions:")
			fmt.Println()

			errorStr := err.Error()

			// Provide specific help based on error type
			if contains(errorStr, "tls: first record does not look like a TLS handshake") {
				fmt.Println("  TLS Handshake Error:")
				fmt.Println("  - This means the port and TLS method don't match")
				fmt.Println("  - For port 587, use STARTTLS (automatic)")
				fmt.Println("  - For port 465, use implicit TLS (automatic)")
				fmt.Println("  - Verify your SMTP provider's documentation")
			} else if contains(errorStr, "554") && contains(errorStr, "DNS PTR") {
				fmt.Println("  DNS PTR Record Error:")
				fmt.Println("  - Your server's IP needs a reverse DNS (PTR) record")
				fmt.Println("  - Contact your hosting provider to set this up")
				fmt.Println("  - PTR should point to mail.yourdomain.com")
				fmt.Println("  - Also ensure you have an SPF record:")
				fmt.Println("    v=spf1 ip4:YOUR.SERVER.IP ~all")
			} else if contains(errorStr, "535") || contains(errorStr, "Authentication failed") {
				fmt.Println("  Authentication Error:")
				fmt.Println("  - Check your EMAIL and SMTP_SECRET are correct")
				fmt.Println("  - For Gmail, use an App Password, not your account password")
				fmt.Println("  - For SendGrid, EMAIL should be literally \"apikey\"")
			} else if contains(errorStr, "550") || contains(errorStr, "Relaying denied") {
				fmt.Println("  Relay Denied Error:")
				fmt.Println("  - Your SMTP server doesn't recognize you as authorized")
				fmt.Println("  - Verify the EMAIL matches an authorized sending address")
				fmt.Println("  - Check your SPF record includes your server IP")
			} else if contains(errorStr, "connection refused") || contains(errorStr, "no such host") {
				fmt.Println("  Connection Error:")
				fmt.Println("  - Cannot connect to SMTP server")
				fmt.Println("  - Verify SMTP hostname is correct")
				fmt.Println("  - Check firewall allows outbound connections on port", smtpPort)
				fmt.Println("  - Try: telnet", smtp, smtpPort)
			} else if contains(errorStr, "timeout") {
				fmt.Println("  Timeout Error:")
				fmt.Println("  - Connection to SMTP server timed out")
				fmt.Println("  - Check firewall allows outbound connections on port", smtpPort)
				fmt.Println("  - Verify SMTP server is reachable")
			}

			fmt.Println()
			fmt.Println("For detailed configuration help, see docs/EMAIL_SETUP.md")
			os.Exit(1)
		}

		fmt.Println("✓ Test email sent successfully!")
		fmt.Println()
		fmt.Println("========================================")
		fmt.Printf("✅ Email configuration is working!\n")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Printf("Check %s for the test message.\n", testEmail)
		fmt.Println("If you don't see it within a few minutes, check your spam folder.")
	},
}

func init() {
	rootCmd.AddCommand(mailtestCmd)
}

// maskIfEmpty returns a masked string or "(not set)" if empty
func maskIfEmpty(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

// maskPassword returns a masked version of the password for display
func maskPassword(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
