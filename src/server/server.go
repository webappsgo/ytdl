// Package server implements the HTTP server with routing and middleware.
// See AI.md PART 1, 12, 13, 14 for complete server specifications.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/casapps/ytdl/src/config"
	"github.com/casapps/ytdl/src/mode"
	"github.com/casapps/ytdl/src/paths"
	"github.com/casapps/ytdl/src/graphql"
	"github.com/casapps/ytdl/src/scheduler"
	"github.com/casapps/ytdl/src/server/handler"
	"github.com/casapps/ytdl/src/server/service"
	"github.com/casapps/ytdl/src/server/store"
	"github.com/casapps/ytdl/src/swagger"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// HTTPServer holds the server configuration and dependencies
type HTTPServer struct {
	Config        *config.ServerConfig
	PathConfig    paths.PathConfig
	BinaryName    string
	Version       string
	CommitID      string
	BuildDate     string
	OfficialSite  string
	Router        chi.Router
	Renderer      *TemplateRenderer
	Store         *store.Store
	AdminStore    *store.AdminStore
	Queue         *service.DownloadQueue
	YTDLPService  *service.YTDLPService
	EmailService  *service.EmailService
	WSHub         *handler.WebSocketHub
	Scheduler     *scheduler.Scheduler
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(cfg *config.ServerConfig, pathConfig paths.PathConfig, binaryName, version, commitID, buildDate, officialSite string) *HTTPServer {
	return &HTTPServer{
		Config:       cfg,
		PathConfig:   pathConfig,
		BinaryName:   binaryName,
		Version:      version,
		CommitID:     commitID,
		BuildDate:    buildDate,
		OfficialSite: officialSite,
	}
}

func (s *HTTPServer) initialize() error {
	// Initialize main database (server.db)
	dbPath := filepath.Join(s.PathConfig.DBDir, "server.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	s.Store = st

	// Initialize admin database (users.db)
	usersDBPath := filepath.Join(s.PathConfig.DBDir, "users.db")
	adminStore, err := store.NewAdminStore(usersDBPath)
	if err != nil {
		return fmt.Errorf("initializing users database: %w", err)
	}
	s.AdminStore = adminStore

	// Initialize yt-dlp service
	s.YTDLPService = service.NewYTDLPService(
		s.Config.Download.YTDLPPath,
		s.Config.Download.FFmpegPath,
	)

	// Initialize download directory
	downloadDir := filepath.Join(s.PathConfig.DataDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	// Initialize download queue (with audio/media processors)
	s.Queue = service.NewDownloadQueue(
		st,
		s.YTDLPService,
		s.Config.Download.FFmpegPath,
		s.Config.Download.Workers,
		downloadDir,
		s.Config.Download.RetentionHours,
	)

	// Initialize WebSocket hub
	s.WSHub = handler.NewWebSocketHub()

	// Connect queue progress to WebSocket broadcast
	s.Queue.OnProgress = s.WSHub.BroadcastProgress

	// Initialize scheduler
	s.Scheduler = scheduler.NewScheduler(st, s.Queue, s.YTDLPService, downloadDir, s.Config.Download.RetentionHours)

	// Initialize template renderer (per PART 16)
	s.Renderer = NewTemplateRenderer()

	// Initialize audit logger (per PART 11)
	auditLogger, err := NewAuditLogger(s.PathConfig.LogDir)
	if err != nil {
		log.Printf("Warning: audit logging disabled: %v", err)
	}
	_ = auditLogger

	// Generate default config on first run (comments above settings per AI.md)
	if _, err := os.Stat(s.PathConfig.ConfigFile); os.IsNotExist(err) {
		log.Printf("First run - generating default configuration at %s", s.PathConfig.ConfigFile)
		if err := config.GenerateDefaultConfig(s.PathConfig.ConfigFile); err != nil {
			log.Printf("Warning: failed to save default config: %v", err)
		}
	}

	// Initialize email service with SMTP auto-detection (per PART 18)
	// MUST be after config generation so detected SMTP can be saved to server.yml (step 4)
	s.EmailService = service.NewEmailService(service.EmailConfig{}, s.Config.Server.FQDN, s.PathConfig.ConfigFile)

	// Start config file watcher for live reload (per PART 5)
	configWatcher := config.NewConfigWatcher(s.PathConfig.ConfigFile, s.Config, func(newCfg *config.ServerConfig) {
		s.Config = newCfg
		log.Println("Configuration reloaded")
	})
	configWatcher.Start()

	// PID file handling
	if s.PathConfig.PIDFile != "" {
		if err := writePIDFile(s.PathConfig.PIDFile); err != nil {
			return fmt.Errorf("PID file: %w", err)
		}
	}

	// Generate setup token on first run (no admins exist)
	if !s.AdminStore.HasAdmins() {
		token, err := s.AdminStore.GenerateSetupToken()
		if err != nil {
			return fmt.Errorf("generating setup token: %w", err)
		}
		s.printSetupBanner(token)
	}

	// Setup router with all dependencies
	s.Router = s.setupRouter()

	return nil
}

func (s *HTTPServer) setupRouter() chi.Router {
	r := chi.NewRouter()

	// Middleware chain - order matters (security first)
	r.Use(chimiddleware.RealIP)
	r.Use(PathSecurityMiddleware)
	r.Use(SecurityHeadersMiddleware)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)

	if mode.IsAppModeDev() {
		r.Use(chimiddleware.Logger)
	}

	// CORS (per PART 14)
	r.Use(CORSMiddleware(DefaultCORSConfig()))

	// Rate limiting (per PART 11)
	rateLimiter := NewRateLimiter(10, 20)
	r.Use(rateLimiter.RateLimitMiddleware)

	// Cache control headers (per PART 9)
	r.Use(CacheControlMiddleware)

	// CSRF protection (per PART 1)
	csrfStore := NewCSRFStore()
	r.Use(CSRFMiddleware(csrfStore))

	// Metrics collection (per PART 21)
	metricsCollector := handler.NewMetricsCollector(s.Store)
	r.Use(metricsCollector.MetricsMiddleware)

	// Health endpoints (both root and API)
	healthHandler := handler.NewHealthHandler(s.Version, s.CommitID, s.BuildDate)
	if s.Store != nil {
		healthHandler.SetDB(s.Store.DB())
	}
	r.Get("/healthz", healthHandler.HandleHealthCheck)

	// WebSocket endpoint
	r.Get("/ws", s.WSHub.HandleWebSocket)

	// Auth routes (not under admin path - shared login endpoint)
	adminHandler := handler.NewAdminHandler(s.AdminStore, s.Config, s.Version, s.Renderer.Render)
	r.Post("/auth/login", adminHandler.HandleLogin)
	r.Get("/auth/logout", adminHandler.HandleLogout)
	r.Post("/auth/logout", adminHandler.HandleLogout)

	// Download handler
	downloadHandler := handler.NewDownloadHandler(s.Store, s.Queue)

	// Search handler
	searchHandler := handler.NewSearchHandler(s.YTDLPService)

	// Library handler
	libraryHandler := handler.NewLibraryHandler(s.Store)

	// Sharing handler
	sharingHandler := handler.NewSharingHandler(s.Store, s.Version, s.OfficialSite)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", healthHandler.HandleHealthCheck)
		r.Get("/version", healthHandler.HandleVersion)

		// Search
		r.Get("/search", searchHandler.HandleSearch)

		// Download endpoints
		r.Post("/downloads", downloadHandler.HandleSubmitDownload)
		r.Post("/downloads/batch", downloadHandler.HandleBatchSubmit)
		r.Get("/downloads", downloadHandler.HandleListDownloads)
		r.Get("/downloads/{id}", downloadHandler.HandleGetDownload)
		r.Delete("/downloads/{id}", downloadHandler.HandleDeleteDownload)
		r.Post("/downloads/{id}/cancel", downloadHandler.HandleCancelDownload)
		r.Post("/downloads/{id}/pause", downloadHandler.HandlePauseDownload)
		r.Post("/downloads/{id}/resume", downloadHandler.HandleResumeDownload)
		r.Post("/downloads/{id}/retry", downloadHandler.HandleRetryDownload)
		r.Get("/downloads/{id}/file", downloadHandler.HandleDownloadFile)
		r.Post("/downloads/{id}/share", sharingHandler.HandleCreateShareLink)
		r.Get("/downloads/{id}/metadata", libraryHandler.HandleGetMetadata)
		r.Put("/downloads/{id}/metadata", libraryHandler.HandleUpdateMetadata)

		// Media library
		r.Get("/library", libraryHandler.HandleBrowseLibrary)

		// Collections
		r.Get("/collections", libraryHandler.HandleListCollections)
		r.Post("/collections", libraryHandler.HandleCreateCollection)
		r.Delete("/collections/{id}", libraryHandler.HandleDeleteCollection)
		r.Post("/collections/{id}/items", libraryHandler.HandleAddToCollection)

		// Presets
		r.Get("/presets", libraryHandler.HandleListPresets)
		r.Post("/presets", libraryHandler.HandleCreatePreset)
		r.Delete("/presets/{id}", libraryHandler.HandleDeletePreset)

		// Watch rules
		r.Get("/watch-rules", libraryHandler.HandleListWatchRules)
		r.Post("/watch-rules", libraryHandler.HandleCreateWatchRule)
		r.Delete("/watch-rules/{id}", libraryHandler.HandleDeleteWatchRule)

		// Analytics
		r.Get("/analytics", libraryHandler.HandleGetAnalytics)

		// RSS/Podcast feed
		r.Get("/feed/rss", sharingHandler.HandleRSSFeed)

		// Browser extension API
		r.Post("/ext/download", sharingHandler.HandleBrowserExtensionSubmit)

		// Admin API routes (authenticated)
		adminPath := s.Config.Server.AdminPath
		r.Route("/"+adminPath, func(r chi.Router) {
			r.Use(adminHandler.AdminAuthMiddleware)
			r.Get("/server/settings", adminHandler.HandleServerSettings)
			r.Patch("/server/settings", adminHandler.HandleServerSettings)
		})
	})

	// Shared download links (public)
	r.Get("/dl/{token}", sharingHandler.HandleShareDownload)

	// Static assets (embedded)
	staticHandler := http.FileServer(http.FS(StaticFS))
	r.Handle("/static/*", staticHandler)

	// PWA manifest
	r.Get("/manifest.json", s.handleManifest)

	// Prometheus metrics (per PART 21)
	r.Get("/metrics", metricsCollector.HandleMetrics)

	// robots.txt and security.txt
	r.Get("/robots.txt", handler.HandleRobotsTxt)
	r.Get("/.well-known/security.txt", handler.HandleSecurityTxt)

	// Swagger/OpenAPI (root-level, not under /api/v1/)
	swaggerHandler := swagger.NewHandler(s.Version, s.OfficialSite)
	r.Get("/openapi", swaggerHandler.HandleUI)
	r.Get("/openapi.json", swaggerHandler.HandleSpec)

	// GraphQL
	graphqlHandler := graphql.NewHandler(s.Store)
	r.HandleFunc("/graphql", graphqlHandler.HandleQuery)
	r.HandleFunc("/api/v1/graphql", graphqlHandler.HandleQuery)

	// Admin panel web routes
	adminPath := s.Config.Server.AdminPath
	r.Get("/"+adminPath, adminHandler.HandleLoginPage)
	r.Route("/"+adminPath, func(r chi.Router) {
		// Setup (no auth required - validates setup token)
		r.Get("/server/setup", adminHandler.HandleSetupPage)
		r.Post("/server/setup", adminHandler.HandleSetupSubmit)

		// Authenticated admin routes
		r.Group(func(r chi.Router) {
			r.Use(adminHandler.AdminAuthMiddleware)
			r.Get("/", adminHandler.HandleDashboard)
			r.Get("/server/settings", adminHandler.HandleServerSettings)
		})
	})

	// Web routes (HTML)
	r.Get("/", s.handleHomePage)

	return r
}

// Start starts the HTTP server with graceful shutdown
func (s *HTTPServer) Start() error {
	if err := s.initialize(); err != nil {
		return err
	}
	defer s.Store.Close()
	defer s.AdminStore.Close()

	// Start WebSocket hub
	go s.WSHub.Run()

	// Start download queue
	s.Queue.Start()
	defer s.Queue.Stop()

	// Start scheduler
	s.Scheduler.Start()
	defer s.Scheduler.Stop()

	addr := fmt.Sprintf("%s:%d", s.Config.Server.Address, s.Config.Server.Port)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.Router,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		s.printStartupBanner(addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	// Remove PID file on shutdown
	if s.PathConfig.PIDFile != "" {
		os.Remove(s.PathConfig.PIDFile)
	}

	log.Println("Server stopped gracefully")
	return nil
}

func (s *HTTPServer) printStartupBanner(addr string) {
	displayVersion := s.Version
	if len(s.Version) > 0 && s.Version[0] >= '0' && s.Version[0] <= '9' && strings.Contains(s.Version, ".") {
		displayVersion = "v" + s.Version
	}

	appMode := "production"
	if mode.IsAppModeDev() {
		appMode = "development"
	}

	log.Println("╭───────────────────────────────────────────────────────────╮")
	log.Printf("│  YTDL · %s                                          │", displayVersion)
	log.Println("├───────────────────────────────────────────────────────────┤")
	log.Printf("│  Running in mode: %s", appMode)
	log.Printf("│  Listening on http://%s", addr)
	log.Printf("│  Database: %s", filepath.Join(s.PathConfig.DBDir, "server.db"))
	log.Printf("│  Downloads: %s", filepath.Join(s.PathConfig.DataDir, "downloads"))
	log.Printf("│  Workers: %d", s.Config.Download.Workers)
	log.Println("╰───────────────────────────────────────────────────────────╯")
}

func (s *HTTPServer) printSetupBanner(setupToken string) {
	log.Println("┌───────────────────────────────────────────────────────────┐")
	log.Println("│  SETUP REQUIRED                                           │")
	log.Println("├───────────────────────────────────────────────────────────┤")
	log.Printf("│  Setup Token: %s", setupToken)
	log.Println("│                                                           │")
	log.Printf("│  Go to /%s/server/setup", s.Config.Server.AdminPath)
	log.Println("│  and enter this token to complete setup.                  │")
	log.Println("│                                                           │")
	log.Println("│  This token will only be shown ONCE.                      │")
	log.Println("└───────────────────────────────────────────────────────────┘")
}

func (s *HTTPServer) handleHomePage(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isAPIRequest(accept) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"ytdl","version":"%s","status":"ok"}`+"\n", s.Version)
		return
	}

	// Serve embedded HTML template
	data, err := TemplateFS.ReadFile("template/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *HTTPServer) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	fmt.Fprint(w, `{
  "name": "ytdl",
  "short_name": "ytdl",
  "description": "Self-hosted media downloader",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#0f0f1a",
  "theme_color": "#e94560",
  "icons": [],
  "share_target": {
    "action": "/api/v1/ext/download",
    "method": "POST",
    "enctype": "application/json",
    "params": {
      "url": "url"
    }
  }
}
`)
}

func isAPIRequest(accept string) bool {
	return accept == "application/json" ||
		accept == "text/json" ||
		(accept != "" && !containsHTML(accept))
}

func containsHTML(accept string) bool {
	return len(accept) > 0 && (strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*"))
}

// writePIDFile writes the current process ID to the PID file
func writePIDFile(pidPath string) error {
	// Check for stale PID
	if data, err := os.ReadFile(pidPath); err == nil {
		var existingPID int
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &existingPID)
		if existingPID > 0 {
			// Check if process is still running
			process, err := os.FindProcess(existingPID)
			if err == nil && process != nil {
				// Process found - assume running (stale detection is best-effort)
				return fmt.Errorf("already running (pid %d) - remove %s if stale", existingPID, pidPath)
			}
			// Stale PID file - remove it
			os.Remove(pidPath)
		}
	}

	// Write our PID
	pid := os.Getpid()
	return os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}
