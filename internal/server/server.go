package server

import (
	"encoding/json"
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
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"initial": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			return string(s[0])
		},
		"empty": func(s interface{}) bool {
			if s == nil {
				return true
			}
			str, ok := s.(string)
			return ok && str == ""
		},
		"ternary": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"contains": func(substr, str string) bool {
			return strings.Contains(str, substr)
		},
		"split": func(sep, str string) []string {
			return strings.Split(str, sep)
		},
		"print": func(args ...interface{}) string {
			return fmt.Sprint(args...)
		},
		"last": func(slice interface{}) interface{} {
			// Handle slice types
			switch s := slice.(type) {
			case []string:
				if len(s) == 0 {
					return ""
				}
				return s[len(s)-1]
			case []interface{}:
				if len(s) == 0 {
					return nil
				}
				return s[len(s)-1]
			}
			return nil
		},
		"CommaInt": func(i interface{}) string {
			// Format integer with comma separators
			switch v := i.(type) {
			case int:
				return fmt.Sprintf("%,d", v)
			case int64:
				return fmt.Sprintf("%,d", v)
			case float64:
				return fmt.Sprintf("%,.0f", v)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b interface{}) bool {
			switch av := a.(type) {
			case int:
				if bv, ok := b.(int); ok {
					return av < bv
				}
			case float64:
				if bv, ok := b.(float64); ok {
					return av < bv
				}
			}
			return false
		},
		"gt": func(a, b interface{}) bool {
			switch av := a.(type) {
			case int:
				if bv, ok := b.(int); ok {
					return av > bv
				}
			case float64:
				if bv, ok := b.(float64); ok {
					return av > bv
				}
			}
			return false
		},
		"le": func(a, b interface{}) bool {
			switch av := a.(type) {
			case int:
				if bv, ok := b.(int); ok {
					return av <= bv
				}
			case float64:
				if bv, ok := b.(float64); ok {
					return av <= bv
				}
			}
			return false
		},
		"ge": func(a, b interface{}) bool {
			switch av := a.(type) {
			case int:
				if bv, ok := b.(int); ok {
					return av >= bv
				}
			case float64:
				if bv, ok := b.(float64); ok {
					return av >= bv
				}
			}
			return false
		},
		"int64": func(i interface{}) int64 {
			switch v := i.(type) {
			case int:
				return int64(v)
			case int64:
				return v
			case float64:
				return int64(v)
			default:
				return 0
			}
		},
		"ToJSON": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return string(b)
		},
		"add1": func(i int) int {
			return i + 1
		},
		"sub1": func(i int) int {
			return i - 1
		},
		"Percent": func(current, total interface{}) string {
			// Convert to float64 for calculation
			var c, t float64
			switch v := current.(type) {
			case int:
				c = float64(v)
			case float64:
				c = v
			default:
				return "0%"
			}
			switch v := total.(type) {
			case int:
				t = float64(v)
			case float64:
				t = v
			default:
				return "0%"
			}
			if t == 0 {
				return "0%"
			}
			return fmt.Sprintf("%.1f%%", (c/t)*100)
		},
		"toString": func(v interface{}) string {
			return fmt.Sprintf("%v", v)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"join": func(sep string, elems []string) string {
			return strings.Join(elems, sep)
		},
		"replace": func(old, new, src string) string {
			return strings.Replace(src, old, new, -1)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
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