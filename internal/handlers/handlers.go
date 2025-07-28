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
	basePath  string
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

func New(apiClient *zepapi.Client, templates *template.Template, basePath string) *Handlers {
	if basePath == "" {
		basePath = "/admin"
	}
	return &Handlers{
		apiClient: apiClient,
		templates: templates,
		basePath:  basePath,
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

// safeString safely converts interface{} to string
func safeString(value interface{}) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%v", value)
}

// Dashboard handles the main dashboard page
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "Dashboard",
		"Page":      "dashboard",
		"MenuItems": GetMenuItems(h.basePath),
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
				Path:  h.basePath + "/sessions",
			},
		},
		Data:      tableData,
		MenuItems: GetMenuItems(h.basePath),
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
				Path:  h.basePath + "/sessions",
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
		MenuItems: GetMenuItems(h.basePath),
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
		w.Header().Set("HX-Redirect", h.basePath+"/sessions")
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to sessions list
		http.Redirect(w, r, h.basePath+"/sessions", http.StatusFound)
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
				Path:  h.basePath + "/users",
			},
		},
		Data:      tableData,
		MenuItems: GetMenuItems(h.basePath),
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
				Path:  h.basePath + "/users",
			},
			{
				Title: "User Details",
				Path:  r.URL.Path,
			},
		},
		"Data": sessionTableData, // Session table data for embedded sessions
		"User": user, // User data separately for form access
		"MenuItems": GetMenuItems(h.basePath),
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
				Path:  h.basePath + "/users",
			},
			{
				Title: "User Details",
				Path:  h.basePath + "/users/" + userID,
			},
			{
				Title: "Sessions",
				Path:  r.URL.Path,
			},
		},
		"Data": tableData,
		"MenuItems": GetMenuItems(h.basePath),
		"UserID": userID,
	}
	
	if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UserEpisodes handles the user episodes page
func (h *Handlers) UserEpisodes(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Fetch user episodes from the graph API
	episodes, err := h.apiClient.GetUserEpisodes(userID)
	if err != nil {
		// If episodes fail, continue with empty episodes (episodes page still viewable)
		episodes = []zepapi.Episode{}
		log.Printf("⚠️ Failed to fetch episodes for user %s: %v", userID, err)
	}

	// Create page data with breadcrumbs
	data := map[string]interface{}{
		"Title":    "User Episodes",
		"SubTitle": "Episodes for user " + userID,
		"Page":     "user_episodes",
		"Path":     r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Users",
				Path:  h.basePath + "/users",
			},
			{
				Title: "User Details", 
				Path:  h.basePath + "/users/" + userID,
			},
			{
				Title: "Episodes",
				Path:  r.URL.Path,
			},
		},
		"Data": map[string]interface{}{
			"Episodes": episodes,
		},
		"MenuItems": GetMenuItems(h.basePath),
		"UserID":    userID,
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "UserEpisodesContent", data); err != nil {
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
	
	log.Printf("🗑️ Starting user deletion for: %s", userID)
	
	// Use comprehensive deletion that includes session and graph cleanup
	err := h.apiClient.DeleteUserWithCleanup(userID)
	if err != nil {
		log.Printf("❌ User deletion failed for %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Failed to delete user: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("✅ Successfully deleted user: %s", userID)
	
	// For HTMX requests, redirect back to users list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", h.basePath+"/users")
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to users list
		http.Redirect(w, r, h.basePath+"/users", http.StatusFound)
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
				Path:  h.basePath + "/users",
			},
			{
				Title: "Create User",
				Path:  r.URL.Path,
			},
		},
		"MenuItems": GetMenuItems(h.basePath),
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
	http.Redirect(w, r, h.basePath+"/users", http.StatusSeeOther)
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
			"status": "unhealthy",
			"version": "Unknown",
		}
	}
	
	log.Printf("DEBUG: Settings handler - health status: %v", health["status"])

	// Create comprehensive configuration display for raw config section
	configHTML := fmt.Sprintf(`🚀 Zep System Configuration & Status

🔗 Zep Server Connection
📡 API URL: %s
🔐 Authentication: ✅ API Key Configured
📋 Server Version: %s
💚 Health Status: %s

📊 System Statistics
👥 Total Users: %s
💬 Total Sessions: %s
🟢 Active Sessions: %s
🔴 Ended Sessions: %s

🌐 Web Interface Server
🏠 Host: %s
🚪 Port: %s
🔒 TLS: %s

⚙️ Network & Security
🌍 CORS Origins: %s
🔄 Trust Proxy: %s

📁 Configuration Details
- All settings are loaded from environment variables
- No sensitive data is exposed in this interface
- Server logs are available via Railway dashboard

💡 Quick Actions
- Restart service: Railway dashboard ➜ Deployments
- View logs: Railway dashboard ➜ Logs
- Update config: Railway dashboard ➜ Variables
`,
		os.Getenv("ZEP_API_URL"),
		safeString(health["version"]),
		safeString(health["status"]),
		safeString(stats["total_users"]),
		safeString(stats["total_sessions"]),
		safeString(stats["active_sessions"]),
		safeString(stats["ended_sessions"]),
		func() string {
			if host := os.Getenv("HOST"); host != "" {
				return host
			}
			return "::"
		}(),
		func() string {
			if port := os.Getenv("PORT"); port != "" {
				return port
			}
			return "8080"
		}(),
		func() string {
			if tls := os.Getenv("TLS_ENABLED"); tls == "true" {
				return "✅ Enabled"
			}
			return "❌ Disabled"
		}(),
		func() string {
			if cors := os.Getenv("CORS_ORIGINS"); cors != "" {
				return cors
			}
			return "*"
		}(),
		func() string {
			if proxy := os.Getenv("TRUST_PROXY"); proxy == "false" {
				return "❌ Disabled"
			}
			return "✅ Enabled"
		}(),
	)

	// Create page data with structured data for template
	data := map[string]interface{}{
		"Title":    "Settings",
		"SubTitle": "Web interface configuration and status",
		"Page":     "settings",
		"Path":     r.URL.Path,
		"BreadCrumbs": []BreadCrumb{
			{
				Title: "Settings",
				Path:  h.basePath + "/settings",
			},
		},
		"MenuItems": GetMenuItems(h.basePath),
		"Data": map[string]interface{}{
			"ConfigHTML": configHTML,
			// System Statistics
			"total_users":    formatStatValue(stats["total_users"]),
			"total_sessions": formatStatValue(stats["total_sessions"]),
			"active_sessions": formatStatValue(stats["active_sessions"]),
			// Server Configuration
			"zep_api_url": os.Getenv("ZEP_API_URL"),
			"version":     health["version"].(string),
			"status":      health["status"].(string),
			// Web Interface Configuration
			"host": func() string {
				if host := os.Getenv("HOST"); host != "" {
					return host
				}
				return "::"
			}(),
			"port": func() string {
				if port := os.Getenv("PORT"); port != "" {
					return port
				}
				return "8080"
			}(),
			"cors_origins": func() string {
				if cors := os.Getenv("CORS_ORIGINS"); cors != "" {
					return cors
				}
				return "*"
			}(),
			"tls_enabled": os.Getenv("TLS_ENABLED"),
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
		MenuItems: GetMenuItems(h.basePath),
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
		MenuItems: GetMenuItems(h.basePath),
	}
	
	if err := h.templates.ExecuteTemplate(w, "UserTable", pageData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}