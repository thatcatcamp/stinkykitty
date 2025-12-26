package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/backup"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage system backups",
	Long:  "Commands for managing system backups: list, restore, delete, and status",
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Run: func(cmd *cobra.Command, args []string) {
		backupPath := "/var/lib/stinkykitty/backups"

		systemDir := filepath.Join(backupPath, "system")
		entries, err := os.ReadDir(systemDir)
		if err != nil {
			log.Fatalf("failed to list backups: %v", err)
		}

		if len(entries) == 0 {
			fmt.Println("No backups found")
			return
		}

		fmt.Println("Available backups:")
		for i, entry := range entries {
			if !entry.IsDir() {
				info, _ := entry.Info()
				size := info.Size()
				modified := info.ModTime().Format("2006-01-02 15:04:05")
				fmt.Printf("%d. %s (%s, %d bytes)\n", i+1, entry.Name(), modified, size)
			}
		}
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <filename>",
	Short: "Restore system from a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		backupPath := "/var/lib/stinkykitty/backups"
		manager := backup.NewBackupManager(backupPath)

		// Confirm restore (safety check)
		fmt.Printf("WARNING: This will overwrite your database and media files.\n")
		fmt.Printf("Are you sure you want to restore from '%s'? (type 'yes' to confirm): ", filename)

		var confirmation string
		fmt.Scanln(&confirmation)
		if confirmation != "yes" {
			fmt.Println("Restore cancelled.")
			return
		}

		// Perform restore
		if err := manager.RestoreBackup(filename); err != nil {
			log.Fatalf("restore failed: %v", err)
		}

		fmt.Printf("Successfully restored from %s\n", filename)
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <filename>",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		backupPath := "/var/lib/stinkykitty/backups"

		backupFile := filepath.Join(backupPath, "system", filename)

		// Confirm deletion
		fmt.Printf("Are you sure you want to delete '%s'? (type 'yes' to confirm): ", filename)

		var confirmation string
		fmt.Scanln(&confirmation)
		if confirmation != "yes" {
			fmt.Println("Deletion cancelled.")
			return
		}

		// Delete backup
		if err := os.Remove(backupFile); err != nil {
			log.Fatalf("failed to delete backup: %v", err)
		}

		fmt.Printf("Successfully deleted %s\n", filename)
	},
}

var backupStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup status and statistics",
	Run: func(cmd *cobra.Command, args []string) {
		backupPath := "/var/lib/stinkykitty/backups"

		systemDir := filepath.Join(backupPath, "system")
		entries, err := os.ReadDir(systemDir)
		if err != nil {
			log.Fatalf("failed to read backup directory: %v", err)
		}

		var totalSize int64
		var backupCount int
		var oldestTime time.Time
		var newestTime time.Time

		for _, entry := range entries {
			if !entry.IsDir() {
				backupCount++
				info, _ := entry.Info()
				totalSize += info.Size()
				modTime := info.ModTime()

				if oldestTime.IsZero() || modTime.Before(oldestTime) {
					oldestTime = modTime
				}
				if newestTime.IsZero() || modTime.After(newestTime) {
					newestTime = modTime
				}
			}
		}

		fmt.Println("Backup Status:")
		fmt.Printf("  Total backups: %d\n", backupCount)
		fmt.Printf("  Total size: %s\n", formatBytes(totalSize))
		if !oldestTime.IsZero() {
			fmt.Printf("  Oldest backup: %s\n", oldestTime.Format("2006-01-02 15:04:05"))
		}
		if !newestTime.IsZero() {
			fmt.Printf("  Newest backup: %s\n", newestTime.Format("2006-01-02 15:04:05"))
		}
	},
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)

	for _, unit := range units {
		if size < 1024.0 {
			return fmt.Sprintf("%.2f %s", size, unit)
		}
		size /= 1024.0
	}

	return fmt.Sprintf("%.2f TB", size)
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupStatusCmd)
}
