package server

import (
	"fmt"
	"html/template"
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

	// Load templates
	templates, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Create handlers
	h := handlers.New(apiClient, templates)

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
	// Health check (always at root)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"zep-web-interface"}`))
	})

	// Static files (always at /static)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

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
			r.Get("/users", h.UserList)
			r.Get("/users/{userId}", h.UserDetails)
			r.Get("/users/{userId}/sessions", h.UserSessions)
			r.Get("/settings", h.Settings)
		})

		// API routes under proxy path
		apiPath := basePath + "/api"
		r.Route(apiPath, func(r chi.Router) {
			r.Get("/sessions", h.SessionListAPI)
			r.Get("/users", h.UserListAPI)
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
			r.Get("/users", h.UserList)
			r.Get("/users/{userId}", h.UserDetails)
			r.Get("/users/{userId}/sessions", h.UserSessions)
			r.Get("/settings", h.Settings)
		})

		// API routes at root level
		r.Route("/api", func(r chi.Router) {
			r.Get("/sessions", h.SessionListAPI)
			r.Get("/users", h.UserListAPI)
		})
	}
}

func loadTemplates() (*template.Template, error) {
	// Load all templates with functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
	})

	// Parse template files
	patterns := []string{
		"web/templates/layouts/*.html",
		"web/templates/pages/*.html",
		"web/templates/components/*.html",
		"web/templates/components/layout/*.html",
		"web/templates/components/content/*.html",
	}

	for _, pattern := range patterns {
		if _, err := tmpl.ParseGlob(pattern); err != nil {
			// Skip if pattern doesn't match any files yet
			continue
		}
	}

	return tmpl, nil
}