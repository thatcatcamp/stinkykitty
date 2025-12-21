package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/handlers"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server operations",
	Long:  "Start, stop, and manage the StinkyKitty HTTP server",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create Gin router
		r := gin.Default()

		// System routes (no site context needed)
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "stinkykitty",
			})
		})

		// Get base domain from config (default to localhost for development)
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "localhost"
		}

		// Get global IP blocklist from config
		var blocklist []string
		// TODO: Load from config when we add security.blocked_ips to config schema

		// Create rate limiter for admin routes
		loginRateLimiter := middleware.NewRateLimiter(5, time.Minute)

		// Site-required routes
		siteGroup := r.Group("/")
		siteGroup.Use(middleware.SiteResolutionMiddleware(db.GetDB(), baseDomain))
		{
			// Public content routes
			siteGroup.GET("/", handlers.ServeHomepage)

			// Admin routes
			adminGroup := siteGroup.Group("/admin")
			adminGroup.Use(
				middleware.IPFilterMiddleware(blocklist),
				middleware.RateLimitMiddleware(loginRateLimiter, "/admin/login"),
			)
			{
				adminGroup.POST("/login", func(c *gin.Context) {
					c.String(200, "Admin login placeholder")
				})
				adminGroup.GET("/dashboard", func(c *gin.Context) {
					c.String(200, "Admin dashboard placeholder")
				})
			}
		}

		httpPort := config.GetString("server.http_port")
		addr := fmt.Sprintf(":%s", httpPort)

		fmt.Printf("Starting StinkyKitty server on %s\n", addr)
		fmt.Printf("Base domain: %s\n", baseDomain)
		if err := r.Run(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)
}
