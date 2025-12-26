package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/users"
	"golang.org/x/term"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  "Create, list, and manage user accounts",
}

var userCreateCmd = &cobra.Command{
	Use:   "create <email>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		email := args[0]

		// Get password from stdin (hidden)
		fmt.Print("Enter password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // Print newline after hidden input
		password := string(passwordBytes)

		user, err := users.CreateUser(db.GetDB(), email, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User created: %s (ID: %d)\n", user.Email, user.ID)
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		userList, err := users.ListUsers(db.GetDB())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing users: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tEMAIL\tCREATED")
		for _, u := range userList {
			fmt.Fprintf(w, "%d\t%s\t%s\n", u.ID, u.Email, u.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		email := args[0]
		user, err := users.GetUserByEmail(db.GetDB(), email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := users.DeleteUser(db.GetDB(), user.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User deleted: %s\n", email)
	},
}

func init() {
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDeleteCmd)
	rootCmd.AddCommand(userCmd)
}

// initSystemDB initializes the system database connection
func initSystemDB() error {
	if err := initConfig(); err != nil {
		return err
	}

	dbType := config.GetString("database.type")
	dbPath := config.GetString("database.path")

	if err := db.InitDB(dbType, dbPath); err != nil {
		return err
	}

	// Initialize FTS index for SQLite databases
	if dbType == "sqlite" {
		if err := db.InitFTSIndex(); err != nil {
			return err
		}
	}

	return nil
}
