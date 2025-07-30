package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/getzep/zep-web-interface/internal/config"
	"github.com/getzep/zep-web-interface/internal/handlers"
	"github.com/getzep/zep-web-interface/internal/zepapi"
)

func New(cfg *config.Config) (*http.Server, error) {
	// Create Zep API client with proxy support
	apiClient := zepapi.NewClient(cfg.ZepAPIURL, cfg.ZepAPIKey, cfg.ProxyURL)

	// Load templates with proxy path support
	templates, err := loadTemplatesWithConfig(cfg.ProxyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Create handlers with base path
	basePath := cfg.ProxyPath
	if basePath == "" {
		basePath = "/admin"
	}
	h := handlers.New(apiClient, templates, basePath)

	// Setup router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	
	// Trust proxy headers if enabled (for Railway, Heroku, etc.)
	if cfg.TrustProxy {
		r.Use(middleware.RealIP)
	}
	
	// CORS with configurable origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Forwarded-For", "X-Real-IP"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes (includes static files)
	setupRoutes(r, h, cfg)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}, nil
}

func setupRoutes(r chi.Router, h *handlers.Handlers, cfg *config.Config) {
	// Debug: Log routing configuration
	fmt.Printf("🔧 PROXY_PATH configuration: '%s'\n", cfg.ProxyPath)
	if cfg.ProxyPath != "" {
		fmt.Printf("📍 Using proxy path routing at: %s\n", cfg.ProxyPath)
	} else {
		fmt.Printf("📍 Using default routing at: /admin\n")
	}
	
	// Health check (always at root)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"zep-web-interface"}`))
	})
	
	// Auth test endpoint
	r.Get("/auth-test", h.TestAuth)

	// Static files - serve at both root and proxy path locations
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))
	r.Handle("/static/*", staticHandler)
	
	// Also serve static files under proxy path if configured
	if cfg.ProxyPath != "" {
		proxyStaticPath := strings.TrimSuffix(cfg.ProxyPath, "/") + "/static/*"
		r.Handle(proxyStaticPath, http.StripPrefix(strings.TrimSuffix(cfg.ProxyPath, "/")+"/static/", http.FileServer(http.Dir("web/static"))))
	}

	// Setup routes based on proxy path configuration
	if cfg.ProxyPath != "" {
		// Normalize proxy path
		basePath := cfg.ProxyPath
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		if strings.HasSuffix(basePath, "/") && basePath != "/" {
			basePath = strings.TrimSuffix(basePath, "/")
		}

		// Direct admin routes at proxy path (PROXY_PATH=/admin means admin routes are AT /admin, not /admin/admin)
		r.Route(basePath, func(r chi.Router) {
			r.Get("/", h.Dashboard)
			r.Get("/sessions", h.SessionList)
			r.Get("/sessions/{sessionId}", h.SessionDetails)
			r.Delete("/sessions/{sessionId}", h.DeleteSession)
			r.Get("/users", h.UserList)
			r.Get("/users/create", h.CreateUserForm)
			r.Post("/users/create", h.CreateUser)
			r.Get("/users/{userId}", h.UserDetails)
			r.Patch("/users/{userId}", h.UpdateUser)
			r.Delete("/users/{userId}", h.DeleteUser)
			r.Get("/users/{userId}/sessions", h.UserSessions)
			r.Get("/users/{userId}/episodes", h.UserEpisodes)
			r.Get("/users/{userId}/graph", h.UserGraph)
			r.Get("/settings", h.Settings)
		})

		// API routes under proxy path
		apiPath := basePath + "/api"
		r.Route(apiPath, func(r chi.Router) {
			r.Get("/sessions", h.SessionListAPI)
			r.Get("/users", h.UserListAPI)
			r.Get("/users/{userId}/episodes", h.UserEpisodesAPI)
			r.Get("/users/{userId}/graph", h.UserGraphAPI)
		})
	} else {
		// Default routes (no proxy path)
		// Redirect root to admin
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin", http.StatusFound)
		})

		// Admin routes at root level
		r.Route("/admin", func(r chi.Router) {
			r.Get("/", h.Dashboard)
			r.Get("/sessions", h.SessionList)
			r.Get("/sessions/{sessionId}", h.SessionDetails)
			r.Delete("/sessions/{sessionId}", h.DeleteSession)
			r.Get("/users", h.UserList)
			r.Get("/users/create", h.CreateUserForm)
			r.Post("/users/create", h.CreateUser)
			r.Get("/users/{userId}", h.UserDetails)
			r.Patch("/users/{userId}", h.UpdateUser)
			r.Delete("/users/{userId}", h.DeleteUser)
			r.Get("/users/{userId}/sessions", h.UserSessions)
			r.Get("/users/{userId}/episodes", h.UserEpisodes)
			r.Get("/users/{userId}/graph", h.UserGraph)
			r.Get("/settings", h.Settings)
		})

		// API routes at root level
		r.Route("/api", func(r chi.Router) {
			r.Get("/sessions", h.SessionListAPI)
			r.Get("/users", h.UserListAPI)
			r.Get("/users/{userId}/episodes", h.UserEpisodesAPI)
			r.Get("/users/{userId}/graph", h.UserGraphAPI)
		})
	}
	
	// Debug: Add a catch-all route to help debug 404s
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("❌ 404 Not Found: %s %s\n", r.Method, r.URL.Path)
		
		// Render proper 404 page using NotFoundContent template
		data := map[string]interface{}{
			"Title": "Page Not Found",
		}
		
		// Try to load templates and render 404 page
		templates, err := loadTemplates()
		if err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}
		
		w.WriteHeader(http.StatusNotFound)
		if err := templates.ExecuteTemplate(w, "NotFoundContent", data); err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
		}
	})
}

