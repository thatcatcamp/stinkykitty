package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/handlers"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/tls"
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
			// Favicon - simple cat SVG
			siteGroup.GET("/favicon.ico", func(c *gin.Context) {
				// Simple SVG cat icon
				svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
					<circle cx="50" cy="50" r="40" fill="#ff9800"/>
					<circle cx="35" cy="45" r="5" fill="#333"/>
					<circle cx="65" cy="45" r="5" fill="#333"/>
					<path d="M 35 60 Q 50 70 65 60" stroke="#333" stroke-width="3" fill="none"/>
					<polygon points="20,30 30,10 35,30" fill="#ff9800"/>
					<polygon points="80,30 70,10 65,30" fill="#ff9800"/>
				</svg>`
				c.Data(200, "image/svg+xml", []byte(svg))
			})

			// Public content routes
			siteGroup.GET("/", handlers.ServeHomepage)

			// Static file serving for uploads (site-specific)
			siteGroup.GET("/uploads/*filepath", handlers.ServeUploadedFile)

			// Admin routes
			adminGroup := siteGroup.Group("/admin")
			adminGroup.Use(middleware.IPFilterMiddleware(blocklist))
			{
				// Login form and submission (no auth required)
				adminGroup.GET("/login", handlers.LoginFormHandler)
				adminGroup.POST("/login", middleware.RateLimitMiddleware(loginRateLimiter, "/admin/login"), handlers.LoginHandler)

				// Admin root - redirect to login
				adminGroup.GET("/", func(c *gin.Context) {
					c.Redirect(302, "/admin/login")
				})

				// Logout route (auth required)
				adminGroup.POST("/logout", auth.RequireAuth(), handlers.LogoutHandler)

				// Protected admin routes (auth required)
				adminGroup.Use(auth.RequireAuth())
				{
					adminGroup.GET("/dashboard", handlers.DashboardHandler)
					adminGroup.GET("/pages", handlers.PagesListHandler)
					adminGroup.GET("/pages/new", handlers.NewPageFormHandler)
					adminGroup.POST("/pages", handlers.CreatePageHandler)
					adminGroup.GET("/pages/:id/edit", handlers.EditPageHandler)
					adminGroup.POST("/pages/:id", handlers.UpdatePageHandler)
					adminGroup.POST("/pages/:id/publish", handlers.PublishPageHandler)
					adminGroup.POST("/pages/:id/unpublish", handlers.UnpublishPageHandler)
					adminGroup.POST("/pages/:id/delete", handlers.DeletePageHandler)
					adminGroup.POST("/pages/:id/blocks", handlers.CreateBlockHandler)
					adminGroup.GET("/pages/:id/blocks/new-image", handlers.NewImageBlockFormHandler)
					adminGroup.GET("/pages/:id/blocks/:block_id/edit", handlers.EditBlockHandler)
					adminGroup.POST("/pages/:id/blocks/:block_id", handlers.UpdateBlockHandler)
					adminGroup.POST("/pages/:id/blocks/:block_id/delete", handlers.DeleteBlockHandler)
					adminGroup.POST("/pages/:id/blocks/:block_id/move-up", handlers.MoveBlockUpHandler)
					adminGroup.POST("/pages/:id/blocks/:block_id/move-down", handlers.MoveBlockDownHandler)
					adminGroup.POST("/upload/image", handlers.UploadImageHandler)
					adminGroup.GET("/menu", handlers.MenuHandler)
					adminGroup.POST("/menu", handlers.CreateMenuItemHandler)
					adminGroup.POST("/menu/:id/delete", handlers.DeleteMenuItemHandler)
					adminGroup.POST("/menu/:id/move-up", handlers.MoveMenuItemUpHandler)
					adminGroup.POST("/menu/:id/move-down", handlers.MoveMenuItemDownHandler)
					adminGroup.GET("/export", handlers.ExportSiteHandler(db.GetDB()))
				}
			}
		}

		// Handle all other routes as potential pages
		r.NoRoute(middleware.SiteResolutionMiddleware(db.GetDB(), baseDomain), handlers.ServePage)

		// Check if TLS is enabled
		if config.GetBool("server.tls_enabled") {
			// Load TLS config
			tlsCfg, err := tls.LoadConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load TLS config: %v\n", err)
				os.Exit(1)
			}

			// Initialize TLS manager
			tlsManager, err := tls.NewManager(db.GetDB(), tlsCfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize TLS manager: %v\n", err)
				os.Exit(1)
			}

			// Add HTTPS redirect middleware
			r.Use(middleware.HTTPSRedirectMiddleware())

			// Start HTTP server (port 80) for ACME challenges + redirects
			httpPort := config.GetString("server.http_port")
			httpAddr := fmt.Sprintf(":%s", httpPort)

			// Channel to signal HTTP server startup status
			httpStarted := make(chan error, 1)

			go func() {
				// Create listener first to catch binding errors immediately
				listener, err := net.Listen("tcp", httpAddr)
				if err != nil {
					httpStarted <- fmt.Errorf("failed to bind HTTP server to %s: %w", httpAddr, err)
					return
				}

				httpStarted <- nil
				fmt.Printf("HTTP server listening on %s (ACME challenges + redirects)\n", httpAddr)

				if err := http.Serve(listener, r); err != nil {
					fmt.Fprintf(os.Stderr, "HTTP server failed: %v\n", err)
					os.Exit(1)
				}
			}()

			// Wait for HTTP server to start (or fail)
			if err := <-httpStarted; err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "Hint: Port 80 typically requires root/sudo privileges\n")
				os.Exit(1)
			}

			// Start HTTPS server (port 443) for main traffic
			httpsPort := config.GetString("server.https_port")
			httpsAddr := fmt.Sprintf(":%s", httpsPort)
			fmt.Printf("Starting HTTPS server on %s\n", httpsAddr)
			fmt.Printf("Base domain: %s\n", baseDomain)

			server := &http.Server{
				Addr:      httpsAddr,
				Handler:   r,
				TLSConfig: tlsManager.GetTLSConfig(),
			}

			if err := server.ListenAndServeTLS("", ""); err != nil {
				fmt.Fprintf(os.Stderr, "HTTPS server error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Dev mode - HTTP only (existing behavior)
			httpPort := config.GetString("server.http_port")
			httpAddr := fmt.Sprintf(":%s", httpPort)
			fmt.Printf("Starting HTTP server on %s (TLS disabled)\n", httpAddr)
			fmt.Printf("Base domain: %s\n", baseDomain)
			if err := r.Run(httpAddr); err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)
}
