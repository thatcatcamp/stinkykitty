package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
	"github.com/thatcatcamp/stinkykitty/internal/users"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage sites",
	Long:  "Create, list, and manage camp sites",
}

var siteCreateCmd = &cobra.Command{
	Use:   "create <subdomain>",
	Short: "Create a new site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		ownerEmail, _ := cmd.Flags().GetString("owner")

		if ownerEmail == "" {
			fmt.Fprintf(os.Stderr, "Error: --owner flag is required\n")
			os.Exit(1)
		}

		// Get owner user
		owner, err := users.GetUserByEmail(db.GetDB(), ownerEmail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: owner user not found: %v\n", err)
			os.Exit(1)
		}

		sitesDir := config.GetString("storage.sites_dir")
		site, err := sites.CreateSite(db.GetDB(), subdomain, owner.ID, sitesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Site created: %s (ID: %d)\n", site.Subdomain, site.ID)
		fmt.Printf("Site directory: %s\n", site.SiteDir)
	},
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		siteList, err := sites.ListSites(db.GetDB())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sites: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSUBDOMAIN\tCUSTOM DOMAIN\tOWNER\tCREATED")
		for _, s := range siteList {
			customDomain := "-"
			if s.CustomDomain != nil {
				customDomain = *s.CustomDomain
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				s.ID, s.Subdomain, customDomain, s.Owner.Email, s.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()
	},
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete <subdomain>",
	Short: "Delete a site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := sites.DeleteSite(db.GetDB(), site.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Site deleted: %s\n", subdomain)
	},
}

var siteAddUserCmd = &cobra.Command{
	Use:   "add-user <subdomain> <email>",
	Short: "Add a user to a site",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		email := args[1]
		role, _ := cmd.Flags().GetString("role")

		if role == "" {
			role = "editor" // default role
		}

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: site not found: %v\n", err)
			os.Exit(1)
		}

		user, err := users.GetUserByEmail(db.GetDB(), email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: user not found: %v\n", err)
			os.Exit(1)
		}

		if err := sites.AddUserToSite(db.GetDB(), site.ID, user.ID, role); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding user to site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added %s to %s with role: %s\n", email, subdomain, role)
	},
}

var siteListUsersCmd = &cobra.Command{
	Use:   "list-users <subdomain>",
	Short: "List users with access to a site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		siteUsers, err := sites.ListSiteUsers(db.GetDB(), site.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMAIL\tROLE")
		for _, su := range siteUsers {
			fmt.Fprintf(w, "%s\t%s\n", su.User.Email, su.Role)
		}
		w.Flush()
	},
}

var siteAddDomainCmd = &cobra.Command{
	Use:   "add-domain <subdomain> <domain>",
	Short: "Add a custom domain to a site",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		domain := args[1]

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := sites.AddCustomDomain(db.GetDB(), site.ID, domain); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added custom domain %s to site %s\n", domain, subdomain)
	},
}

func init() {
	siteCreateCmd.Flags().String("owner", "", "Email of the site owner (required)")
	siteAddUserCmd.Flags().String("role", "editor", "User role (owner, admin, editor)")

	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteAddUserCmd)
	siteCmd.AddCommand(siteListUsersCmd)
	siteCmd.AddCommand(siteAddDomainCmd)
	rootCmd.AddCommand(siteCmd)
}
