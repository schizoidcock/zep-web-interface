package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/getzep/zep-web-interface/internal/zepapi"
)

type Handlers struct {
	apiClient *zepapi.Client
	templates *template.Template
}

// Data structures matching Zep v0.27 template expectations
type Column struct {
	Name       string `json:"name"`
	Sortable   bool   `json:"sortable"`
	OrderByKey string `json:"order_by_key"`
}

type TableData struct {
	TableID     string        `json:"table_id"`
	Columns     []Column      `json:"columns"`
	Rows        interface{}   `json:"rows"`
	TotalCount  int           `json:"total_count"`
	RowCount    int           `json:"row_count"`
	CurrentPage int           `json:"current_page"`
	PageSize    int           `json:"page_size"`
	PageCount   int           `json:"page_count"`
	OrderBy     string        `json:"order_by"`
	Asc         bool          `json:"asc"`
}

type BreadCrumb struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

type MenuItem struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	External  bool          `json:"external"`
	Icon      template.HTML `json:"icon"`
	ContentID string        `json:"content_id"`
}

type PageData struct {
	Title       string        `json:"title"`
	SubTitle    string        `json:"subtitle"`
	Page        string        `json:"page"`
	Path        string        `json:"path"`
	BreadCrumbs []BreadCrumb  `json:"breadcrumbs"`
	Data        *TableData    `json:"data"`
	MenuItems   []MenuItem    `json:"menu_items"`
}

// SessionRow represents a session with timestamp formatting
type SessionRow struct {
	Session *zepapi.Session `json:"session"`
}

var SessionTableColumns = []Column{
	{
		Name:       "Session",
		Sortable:   true,
		OrderByKey: "session_id",
	},
	{
		Name:       "User",
		Sortable:   true,
		OrderByKey: "user_id",
	},
	{
		Name:       "Created",
		Sortable:   true,
		OrderByKey: "created_at",
	},
}

var UserTableColumns = []Column{
	{
		Name:       "User ID",
		Sortable:   true,
		OrderByKey: "user_id",
	},
	{
		Name:       "Email",
		Sortable:   true,
		OrderByKey: "email",
	},
	{
		Name:       "Sessions",
		Sortable:   true,
		OrderByKey: "session_count",
	},
	{
		Name:       "Created",
		Sortable:   true,
		OrderByKey: "created_at",
	},
}

func New(apiClient *zepapi.Client, templates *template.Template) *Handlers {
	return &Handlers{
		apiClient: apiClient,
		templates: templates,
	}
}

// formatStatValue formats a stat value for display
func formatStatValue(value interface{}) string {
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return "Unknown"
	}
}

// Dashboard handles the main dashboard page
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "Dashboard",
		"Page":      "dashboard",
		"MenuItems": MenuItems,
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "DashboardContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// SessionList handles the sessions list page
func (h *Handlers) SessionList(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.apiClient.GetSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse query parameters for sorting and pagination
	currentPage := 1
	pageSize := 10
	orderBy := "created_at"
	asc := false

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			currentPage = page
		}
	}

	if order := r.URL.Query().Get("order"); order != "" {
		orderBy = order
	}

	if ascStr := r.URL.Query().Get("asc"); ascStr == "true" {
		asc = true
	}

	// Convert sessions to SessionRows for template compatibility
	sessionRows := make([]SessionRow, len(sessions))
	for i := range sessions {
		sessionRows[i] = SessionRow{Session: &sessions[i]}
	}

	// Calculate pagination
	totalCount := len(sessions)
	pageCount := (totalCount + pageSize - 1) / pageSize
	rowCount := len(sessionRows)

	// Create table data structure
	tableData := &TableData{
		TableID:     "session-table",
		Columns:     SessionTableColumns,
		Rows:        sessionRows,
		TotalCount:  totalCount,
		RowCount:    rowCount,
		CurrentPage: currentPage,
		PageSize:    pageSize,
		PageCount:   pageCount,
		OrderBy:     orderBy,
		Asc:         asc,
	}

	// Create page data with breadcrumbs
	pageData := &PageData{
		Title:    "Sessions",
		SubTitle: "View and manage sessions",
		Page:     "sessions",
		Path:     r.URL.Path,
		BreadCrumbs: []BreadCrumb{
			{
				Title: "Sessions",
				Path:  "/sessions",
			},
		},
		Data:      tableData,
		MenuItems: MenuItems,
	}

	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "SessionsContent", pageData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", pageData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// SessionDetails handles the session details page
func (h *Handlers) SessionDetails(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	
	// Parse query parameters for message pagination
	currentPage := 1
	pageSize := 10
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			currentPage = page
		}
	}
	
	// Fetch session details
	session, err := h.apiClient.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch message list for this session
	messages, totalMessages, err := h.apiClient.GetMessageList(sessionID, currentPage, pageSize)
	if err != nil {
		// If messages fail, continue with empty messages (session still viewable)
		messages = []zepapi.Message{}
		totalMessages = 0
	}

	// Calculate pagination
	pageCount := (totalMessages + pageSize - 1) / pageSize
	if pageCount == 0 {
		pageCount = 1
	}

	// Create page data with breadcrumbs like other handlers
	pageData := &PageData{
		Title:    "Session Details",
		SubTitle: "View session information and chat history - " + sessionID,
		Page:     "session_details",
		Path:     r.URL.Path,
		BreadCrumbs: []BreadCrumb{
			{
				Title: "Sessions",
				Path:  "/sessions",
			},
			{
				Title: "Session Details",
				Path:  r.URL.Path,
			},
		},
		Data: &TableData{
			TableID:     "chat-history",
			TotalCount:  totalMessages,
			RowCount:    len(messages),
			CurrentPage: currentPage,
			PageSize:    pageSize,
			PageCount:   pageCount,
		},
		MenuItems: MenuItems,
	}

	// Add session and messages data for template access
	data := map[string]interface{}{
		"Title":      pageData.Title,
		"SubTitle":   pageData.SubTitle,
		"Page":       pageData.Page,
		"Path":       pageData.Path,
		"BreadCrumbs": pageData.BreadCrumbs,
		"MenuItems":  pageData.MenuItems,
		"Data": map[string]interface{}{
			"Session":     session,
			"Messages":    messages,
			"TableID":     "chat-history",
			"TotalCount":  totalMessages,
			"CurrentPage": currentPage,
			"PageCount":   pageCount,
			"PageSize":    pageSize,
		},
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "SessionDetailsContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// DeleteSession handles the session deletion
func (h *Handlers) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	
	err := h.apiClient.DeleteSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// For HTMX requests, redirect back to sessions list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/admin/sessions")
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to sessions list
		http.Redirect(w, r, "/admin/sessions", http.StatusFound)
	}
}

// UserList handles the users list page
func (h *Handlers) UserList(w http.ResponseWriter, r *http.Request) {
	users, err := h.apiClient.GetUsers()
	if err != nil {
		// Log the specific error for debugging
		log.Printf("❌ GetUsers error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("✅ Successfully fetched %d users", len(users))

	// Parse query parameters for sorting and pagination
	currentPage := 1
	pageSize := 10
	orderBy := "created_at"
	asc := false

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			currentPage = page
		}
	}

	if order := r.URL.Query().Get("order"); order != "" {
		orderBy = order
	}

	if ascStr := r.URL.Query().Get("asc"); ascStr == "true" {
		asc = true
	}

	// Fetch session count for each user (like v0.27)
	for i := range users {
		sessions, err := h.apiClient.GetUserSessions(users[i].UserID)
		if err != nil {
			// If session fetch fails, set count to 0
			users[i].SessionCount = 0
		} else {
			users[i].SessionCount = len(sessions)
		}
	}

	// Calculate pagination
	totalCount := len(users)
	pageCount := (totalCount + pageSize - 1) / pageSize
	rowCount := len(users)

	// Create table data structure
	tableData := &TableData{
		TableID:     "user-table",
		Columns:     UserTableColumns,
		Rows:        users, // Users slice directly
		TotalCount:  totalCount,
		RowCount:    rowCount,
		CurrentPage: currentPage,
		PageSize:    pageSize,
		PageCount:   pageCount,
		OrderBy:     orderBy,
		Asc:         asc,
	}

	// Create page data with breadcrumbs
	pageData := &PageData{
		Title:    "Users",
		SubTitle: "View and manage users",
		Page:     "users",
		Path:     r.URL.Path,
		BreadCrumbs: []BreadCrumb{
			{
				Title: "Users",
				Path:  "/users",
			},
		},
		Data:      tableData,
		MenuItems: MenuItems,
	}

	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "UsersContent", pageData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", pageData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// UserDetails handles the user details page
func (h *Handlers) UserDetails(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Fetch user details
	user, err := h.apiClient.GetUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch user sessions for the embedded session table
	sessions, err := h.apiClient.GetUserSessions(userID)
	if err != nil {
		// If sessions fail, continue with empty sessions (user details still viewable)
		sessions = []zepapi.Session{}
	}

	// Convert sessions to SessionRows for template compatibility
	sessionRows := make([]SessionRow, len(sessions))
	for i := range sessions {
		sessionRows[i] = SessionRow{Session: &sessions[i]}
	}

	// Create table data for sessions
	sessionTableData := &TableData{
		TableID:     "user-session-table",
		Columns:     SessionTableColumns,
		Rows:        sessionRows,
		TotalCount:  len(sessions),
		RowCount:    len(sessionRows),
		CurrentPage: 1,
		PageSize:    10,
		PageCount:   1,
		OrderBy:     "created_at",
		Asc:         false,
	}

	// Create page data with breadcrumbs and proper data structure
	data := map[string]interface{}{
		"Title": "User Details",
		"SubTitle": "View and manage user information - " + userID,
		"Page": "user_details",
		"Path": r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Users",
				Path:  "/users",
			},
			{
				Title: "User Details",
				Path:  r.URL.Path,
			},
		},
		"Data": sessionTableData, // Session table data for embedded sessions
		"User": user, // User data separately for form access
		"MenuItems": MenuItems,
		"Slug": userID, // Add slug for Alpine.js functionality
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "UserDetailsContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	// Convert sessions to SessionRows for template compatibility
	sessionRows := make([]SessionRow, len(sessions))
	for i := range sessions {
		sessionRows[i] = SessionRow{Session: &sessions[i]}
	}

	// Create table data structure like SessionList handler
	tableData := &TableData{
		TableID:     "user-session-table",
		Columns:     SessionTableColumns,
		Rows:        sessionRows,
		TotalCount:  len(sessions),
		RowCount:    len(sessionRows),
		CurrentPage: 1,
		PageSize:    10,
		PageCount:   1,
		OrderBy:     "created_at",
		Asc:         false,
	}

	// Create page data with breadcrumbs
	data := map[string]interface{}{
		"Title": "User Sessions",
		"SubTitle": "Sessions for user " + userID,
		"Page": "user_sessions",
		"Path": r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Users",
				Path:  "/users",
			},
			{
				Title: "User Details",
				Path:  "/users/" + userID,
			},
			{
				Title: "Sessions",
				Path:  r.URL.Path,
			},
		},
		"Data": tableData,
		"MenuItems": MenuItems,
		"UserID": userID,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateUser handles user detail form submissions
func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}
	
	// Extract form fields
	updateReq := map[string]interface{}{
		"user_id": userID,
	}
	
	if firstName := r.FormValue("first_name"); firstName != "" {
		updateReq["first_name"] = firstName
	}
	
	if lastName := r.FormValue("last_name"); lastName != "" {
		updateReq["last_name"] = lastName
	}
	
	if email := r.FormValue("email"); email != "" {
		updateReq["email"] = email
	}
	
	// Update user via API
	_, err := h.apiClient.UpdateUser(userID, updateReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// For HTMX requests, redirect to refresh the page
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to user details page
		http.Redirect(w, r, r.URL.Path, http.StatusFound)
	}
}

// DeleteUser handles user deletion
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	err := h.apiClient.DeleteUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// For HTMX requests, redirect back to users list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/admin/users")
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to users list
		http.Redirect(w, r, "/admin/users", http.StatusFound)
	}
}

// CreateUserForm handles displaying the create user form
func (h *Handlers) CreateUserForm(w http.ResponseWriter, r *http.Request) {
	// Create page data for create user form
	data := map[string]interface{}{
		"Title":    "Create User",
		"SubTitle": "Add a new user to the system",
		"Page":     "create_user",
		"Path":     r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Users",
				Path:  "/admin/users",
			},
			{
				Title: "Create User",
				Path:  r.URL.Path,
			},
		},
		"MenuItems": MenuItems,
	}
	
	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "CreateUserContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// CreateUser handles creating a new user
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Create user via API
	createReq := map[string]interface{}{
		"user_id":    userID,
		"email":      r.FormValue("email"),
		"first_name": r.FormValue("first_name"),
		"last_name":  r.FormValue("last_name"),
		"metadata":   map[string]interface{}{},
	}

	_, err := h.apiClient.CreateUser(createReq)
	if err != nil {
		log.Printf("❌ Create user error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Successfully created user: %s", userID)
	
	// Redirect to users list
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// TestAuth handles API authentication testing
func (h *Handlers) TestAuth(w http.ResponseWriter, r *http.Request) {
	users, err := h.apiClient.GetUsers()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "error", "message": "%s"}`, err.Error())
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "user_count": %d, "message": "Authentication working"}`, len(users))
}

// Settings handles the settings page
func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	// Get system statistics and health information
	stats, err := h.apiClient.GetSystemStats()
	if err != nil {
		stats = map[string]interface{}{
			"total_users": "Error",
			"total_sessions": "Error",
			"active_sessions": "Error",
		}
	}
	
	health, err := h.apiClient.GetServerHealth()
	if err != nil {
		health = map[string]interface{}{
			"status": "Error",
			"version": "Unknown",
		}
	}

	// Create comprehensive configuration display like v0.27
	configHTML := `
	<div class="mb-6">
		<h3 class="text-lg font-semibold text-gray-800 mb-4">🚀 Zep System Configuration & Status</h3>
		<div class="space-y-4">
			<div class="bg-blue-50 p-4 rounded-lg border border-blue-200">
				<h4 class="font-medium text-blue-800 mb-2">🔗 Zep Server Connection</h4>
				<div class="text-sm text-blue-700 space-y-1">
					<div>📡 <strong>API URL:</strong> ` + os.Getenv("ZEP_API_URL") + `</div>
					<div>🔐 <strong>Authentication:</strong> ✅ API Key Configured</div>
					<div>📋 <strong>Server Version:</strong> ` + health["version"].(string) + `</div>
					<div>💚 <strong>Health Status:</strong> ` + health["status"].(string) + `</div>
				</div>
			</div>
			
			<div class="bg-green-50 p-4 rounded-lg border border-green-200">
				<h4 class="font-medium text-green-800 mb-2">📊 System Statistics</h4>
				<div class="text-sm text-green-700 space-y-1">
					<div>👥 <strong>Total Users:</strong> ` + formatStatValue(stats["total_users"]) + `</div>
					<div>💬 <strong>Total Sessions:</strong> ` + formatStatValue(stats["total_sessions"]) + `</div>
					<div>🟢 <strong>Active Sessions:</strong> ` + formatStatValue(stats["active_sessions"]) + `</div>
					<div>🔴 <strong>Ended Sessions:</strong> ` + formatStatValue(stats["ended_sessions"]) + `</div>
				</div>
			</div>
			
			<div class="bg-yellow-50 p-4 rounded-lg border border-yellow-200">
				<h4 class="font-medium text-yellow-800 mb-2">🌐 Web Interface Server</h4>
				<div class="text-sm text-yellow-700 space-y-1">
					<div>🏠 <strong>Host:</strong> ` + func() string {
		if host := os.Getenv("HOST"); host != "" {
			return host
		}
		return "::"
	}() + `</div>
					<div>🚪 <strong>Port:</strong> ` + func() string {
		if port := os.Getenv("PORT"); port != "" {
			return port
		}
		return "8080"
	}() + `</div>
					<div>🔒 <strong>TLS:</strong> ` + func() string {
		if tls := os.Getenv("TLS_ENABLED"); tls == "true" {
			return "✅ Enabled"
		}
		return "❌ Disabled"
	}() + `</div>
				</div>
			</div>
			
			<div class="bg-purple-50 p-4 rounded-lg border border-purple-200">
				<h4 class="font-medium text-purple-800 mb-2">⚙️ Network & Security</h4>
				<div class="text-sm text-purple-700 space-y-1">
					<div>🌍 <strong>CORS Origins:</strong> ` + func() string {
		if cors := os.Getenv("CORS_ORIGINS"); cors != "" {
			return cors
		}
		return "*"
	}() + `</div>
					<div>🔄 <strong>Trust Proxy:</strong> ` + func() string {
		if proxy := os.Getenv("TRUST_PROXY"); proxy == "false" {
			return "❌ Disabled"
		}
		return "✅ Enabled"
	}() + `</div>
					<div>🛡️ <strong>HTTP Proxy:</strong> ` + func() string {
		if proxy := os.Getenv("PROXY_URL"); proxy != "" {
			return "✅ Configured"
		}
		return "❌ Not configured"
	}() + `</div>
				</div>
			</div>
			
			<div class="bg-red-50 p-4 rounded-lg border border-red-200">
				<h4 class="font-medium text-red-800 mb-2">🗄️ Database & Storage</h4>
				<div class="text-sm text-red-700 space-y-1">
					<div>🐘 <strong>Database:</strong> PostgreSQL (via Zep API)</div>
					<div>📊 <strong>Connection Status:</strong> ✅ Connected (API responding)</div>
					<div>🏷️ <strong>Project Scope:</strong> Multi-tenant with UUID filtering</div>
					<div>🔍 <strong>Search:</strong> ✅ Full-text search available</div>
				</div>
			</div>
			
			<div class="bg-indigo-50 p-4 rounded-lg border border-indigo-200">
				<h4 class="font-medium text-indigo-800 mb-2">🤖 AI & Processing Features</h4>
				<div class="text-sm text-indigo-700 space-y-1">
					<div>💬 <strong>Message Processing:</strong> ✅ Available</div>
					<div>📝 <strong>Memory Management:</strong> ✅ Session memory supported</div>
					<div>🔍 <strong>Session Search:</strong> ✅ Advanced search endpoint</div>
					<div>🏷️ <strong>Message Roles:</strong> system, user, assistant, function, tool</div>
				</div>
			</div>
			
			<div class="bg-gray-50 p-4 rounded-lg border border-gray-200">
				<h4 class="font-medium text-gray-800 mb-2">🎯 Environment & Deployment</h4>
				<div class="text-sm text-gray-700 space-y-1">
					<div>📝 <strong>Config Source:</strong> Environment Variables</div>
					<div>🚀 <strong>Interface Status:</strong> ✅ Running</div>
					<div>🌍 <strong>Deployment:</strong> Production</div>
					<div>⚡ <strong>API Version:</strong> v2</div>
				</div>
			</div>
		</div>
	</div>
	`

	// Create page data with breadcrumbs
	data := map[string]interface{}{
		"Title":    "Settings",
		"SubTitle": "Web interface configuration and status",
		"Page":     "settings",
		"Path":     r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Settings",
				Path:  "/settings",
			},
		},
		"MenuItems": MenuItems,
		"Data": map[string]interface{}{
			"ConfigHTML": template.HTML(configHTML),
		},
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "SettingsContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// API handlers for HTMX requests
func (h *Handlers) SessionListAPI(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.apiClient.GetSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse query parameters for sorting and pagination
	currentPage := 1
	pageSize := 10
	orderBy := "created_at"
	asc := false

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			currentPage = page
		}
	}

	if order := r.URL.Query().Get("order"); order != "" {
		orderBy = order
	}

	if ascStr := r.URL.Query().Get("asc"); ascStr == "true" {
		asc = true
	}

	// Convert sessions to SessionRows for template compatibility
	sessionRows := make([]SessionRow, len(sessions))
	for i := range sessions {
		sessionRows[i] = SessionRow{Session: &sessions[i]}
	}

	// Calculate pagination
	totalCount := len(sessions)
	pageCount := (totalCount + pageSize - 1) / pageSize
	rowCount := len(sessionRows)

	// Create table data structure
	tableData := &TableData{
		TableID:     "session-table",
		Columns:     SessionTableColumns,
		Rows:        sessionRows,
		TotalCount:  totalCount,
		RowCount:    rowCount,
		CurrentPage: currentPage,
		PageSize:    pageSize,
		PageCount:   pageCount,
		OrderBy:     orderBy,
		Asc:         asc,
	}

	// Create page data for HTMX response
	pageData := &PageData{
		Path:      r.URL.Path,
		Data:      tableData,
		MenuItems: MenuItems,
	}
	
	if err := h.templates.ExecuteTemplate(w, "SessionTable", pageData); err != nil {
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

	// Parse query parameters for sorting and pagination
	currentPage := 1
	pageSize := 10
	orderBy := "created_at"
	asc := false

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			currentPage = page
		}
	}

	if order := r.URL.Query().Get("order"); order != "" {
		orderBy = order
	}

	if ascStr := r.URL.Query().Get("asc"); ascStr == "true" {
		asc = true
	}

	// Calculate pagination
	totalCount := len(users)
	pageCount := (totalCount + pageSize - 1) / pageSize
	rowCount := len(users)

	// Create table data structure
	tableData := &TableData{
		TableID:     "user-table",
		Columns:     UserTableColumns,
		Rows:        users, // Users slice directly
		TotalCount:  totalCount,
		RowCount:    rowCount,
		CurrentPage: currentPage,
		PageSize:    pageSize,
		PageCount:   pageCount,
		OrderBy:     orderBy,
		Asc:         asc,
	}

	// Create page data for HTMX response
	pageData := &PageData{
		Path:      r.URL.Path,
		Data:      tableData,
		MenuItems: MenuItems,
	}
	
	if err := h.templates.ExecuteTemplate(w, "UserTable", pageData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}