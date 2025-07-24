package handlers

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/getzep/zep-web-interface/internal/zepapi"
)

type Handlers struct {
	apiClient *zepapi.Client
	templates *template.Template
}

func New(apiClient *zepapi.Client, templates *template.Template) *Handlers {
	return &Handlers{
		apiClient: apiClient,
		templates: templates,
	}
}

// Dashboard handles the main dashboard page
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Dashboard",
		"Page":  "dashboard",
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SessionList handles the sessions list page
func (h *Handlers) SessionList(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.apiClient.GetSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Sessions",
		"Page":     "sessions",
		"Sessions": sessions,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SessionDetails handles the session details page
func (h *Handlers) SessionDetails(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	
	session, err := h.apiClient.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "Session Details",
		"Page":    "session_details",
		"Session": session,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UserList handles the users list page
func (h *Handlers) UserList(w http.ResponseWriter, r *http.Request) {
	users, err := h.apiClient.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Users",
		"Page":  "users",
		"Users": users,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UserDetails handles the user details page
func (h *Handlers) UserDetails(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	user, err := h.apiClient.GetUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "User Details",
		"Page":  "user_details",
		"User":  user,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UserSessions handles the user sessions page
func (h *Handlers) UserSessions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	sessions, err := h.apiClient.GetUserSessions(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "User Sessions",
		"Page":     "user_sessions",
		"UserID":   userID,
		"Sessions": sessions,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Settings handles the settings page
func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Settings",
		"Page":  "settings",
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// API handlers for HTMX requests
func (h *Handlers) SessionListAPI(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.apiClient.GetSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Sessions": sessions,
	}
	
	if err := h.templates.ExecuteTemplate(w, "SessionTable", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) UserListAPI(w http.ResponseWriter, r *http.Request) {
	users, err := h.apiClient.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Users": users,
	}
	
	if err := h.templates.ExecuteTemplate(w, "UserTable", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}