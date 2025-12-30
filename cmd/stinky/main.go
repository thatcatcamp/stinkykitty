// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "stinky",
	Short: "StinkyKitty - Multi-tenant CMS for Burning Man camps",
	Long: `StinkyKitty is a multi-tenant CMS platform designed to replace WordPress
for Burning Man camps and similar community groups.

It provides rich, professional-looking websites while avoiding the security
and maintenance nightmares of plugin ecosystems.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
