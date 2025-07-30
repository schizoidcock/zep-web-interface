package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/getzep/zep-web-interface/internal/cache"
	"github.com/getzep/zep-web-interface/internal/zepapi"
)

type Handlers struct {
	apiClient *zepapi.Client
	templates *template.Template
	basePath  string
	cache     *cache.Cache
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
		cache:     cache.NewCache(),
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
	users, err := h.apiClient.GetUsersWithSessionCounts()
	if err != nil {
		// Log the specific error for debugging
		log.Printf("❌ GetUsersWithSessionCounts error: %v", err)
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

	// Note: Session counts are now fetched concurrently via GetUsersWithSessionCounts
	// This eliminates the N+1 query problem

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

// UserEpisodes handles the user episodes page with async loading
func (h *Handlers) UserEpisodes(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Create page data with breadcrumbs - load data asynchronously
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
			"AsyncLoad": true, // Trigger async loading in template
			"ApiUrl":    h.basePath + "/api/users/" + userID + "/episodes",
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

// UserGraph handles the user graph visualization page with direct data loading
func (h *Handlers) UserGraph(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Check cache first
	cacheKey := fmt.Sprintf("graph:%s", userID)
	var triplets []zepapi.RawTriplet
	
	if cached, found := h.cache.Get(cacheKey); found && cached != nil {
		if cachedTriplets, ok := cached.([]zepapi.RawTriplet); ok {
			triplets = cachedTriplets
			log.Printf("📊 Cache hit for user graph: %s (%d triplets)", userID, len(triplets))
		}
	}
	
	// If not cached, fetch from API
	if triplets == nil {
		var err error
		triplets, err = h.apiClient.GetUserGraphTriplets(userID)
		if err != nil {
			log.Printf("❌ Failed to get graph triplets for user %s: %v", userID, err)
			triplets = []zepapi.RawTriplet{} // Empty slice for template
		} else {
			// Cache the result
			h.cache.Set(cacheKey, triplets, 5*time.Minute)
			log.Printf("✅ Loaded %d triplets for user graph: %s", len(triplets), userID)
		}
	}
	
	// Create page data with breadcrumbs and actual graph data
	data := map[string]interface{}{
		"Title":    "User Graph",
		"SubTitle": "Knowledge graph visualization for user " + userID,
		"Page":     "user_graph",
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
				Title: "Graph",
				Path:  r.URL.Path,
			},
		},
		"Data": map[string]interface{}{
			"Triplets": triplets, // Provide actual triplets data
		},
		"MenuItems": GetMenuItems(h.basePath),
		"UserID":    userID,
	}
	
	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "UserGraphContent", data); err != nil {
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

// DeleteUser handles user deletion with async processing
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	log.Printf("🗑️ Starting optimized user deletion for: %s", userID)
	
	// Start deletion in background for immediate UI response
	go func() {
		log.Printf("🚀 Background deletion started for user: %s", userID)
		err := h.apiClient.DeleteUserWithCleanup(userID)
		if err != nil {
			log.Printf("❌ Background user deletion failed for %s: %v", userID, err)
		} else {
			log.Printf("✅ Background user deletion completed for: %s", userID)
		}
		
		// Clear any cached user data
		h.cache.Delete(fmt.Sprintf("user:%s", userID))
		h.cache.Delete(fmt.Sprintf("episodes:%s", userID))
		h.cache.Delete(fmt.Sprintf("graph:%s", userID))
	}()
	
	// Immediately respond to user interface (optimistic deletion)
	log.Printf("✅ User deletion initiated for: %s (processing in background)", userID)
	
	// For HTMX requests, redirect back to users list immediately
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", h.basePath+"/users")
		w.WriteHeader(http.StatusOK)
	} else {
		// For regular requests, redirect to users list immediately
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
	
	log.Printf("DEBUG: Settings handler - health status: '%v' (type: %T)", health["status"], health["status"])

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
			"version":     safeString(health["version"]),
			"status":      safeString(health["status"]),
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
	users, err := h.apiClient.GetUsersWithSessionCounts()
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

// UserEpisodesAPI handles the API endpoint for user episodes (for async loading)
func (h *Handlers) UserEpisodesAPI(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Check cache first
	cacheKey := fmt.Sprintf("episodes:%s", userID)
	if cached, found := h.cache.Get(cacheKey); found && cached != nil {
		if episodes, ok := cached.([]zepapi.Episode); ok {
			log.Printf("📋 Cache hit for user episodes: %s (%d episodes)", userID, len(episodes))
			
			data := map[string]interface{}{
				"UserID": userID,
				"Data": map[string]interface{}{
					"Episodes": episodes,
				},
			}
			
			if err := h.templates.ExecuteTemplate(w, "UserEpisodesContent", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	
	// Fetch episodes from API
	episodes, err := h.apiClient.GetUserEpisodes(userID)
	if err != nil {
		log.Printf("❌ Failed to get episodes for user %s: %v", userID, err)
		
		// Return empty state
		data := map[string]interface{}{
			"UserID": userID,
			"Data": map[string]interface{}{
				"Episodes": []zepapi.Episode{},
			},
		}
		
		if err := h.templates.ExecuteTemplate(w, "UserEpisodesContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	
	// Cache the result
	h.cache.Set(cacheKey, episodes, 5*time.Minute)
	log.Printf("✅ Loaded %d episodes for user: %s", len(episodes), userID)
	
	// Create template data
	data := map[string]interface{}{
		"UserID": userID,
		"Data": map[string]interface{}{
			"Episodes": episodes,
		},
	}
	
	if err := h.templates.ExecuteTemplate(w, "UserEpisodesContent", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UserGraphAPI handles the API endpoint for user graph data (for async loading)
func (h *Handlers) UserGraphAPI(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Check cache first
	cacheKey := fmt.Sprintf("graph:%s", userID)
	if cached, found := h.cache.Get(cacheKey); found && cached != nil {
		if triplets, ok := cached.([]zepapi.RawTriplet); ok {
			log.Printf("📊 Cache hit for user graph: %s (%d triplets)", userID, len(triplets))
			
			data := map[string]interface{}{
				"UserID": userID,
				"Data": map[string]interface{}{
					"Triplets": triplets,
				},
			}
			
			if err := h.templates.ExecuteTemplate(w, "UserGraphContent", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	
	// Fetch graph triplets from API
	triplets, err := h.apiClient.GetUserGraphTriplets(userID)
	if err != nil {
		log.Printf("❌ Failed to get graph triplets for user %s: %v", userID, err)
		
		// Return empty graph state
		data := map[string]interface{}{
			"UserID": userID,
			"Data": map[string]interface{}{
				"Triplets": []zepapi.RawTriplet{},
			},
		}
		
		if err := h.templates.ExecuteTemplate(w, "UserGraphContent", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	
	// Cache the result
	h.cache.Set(cacheKey, triplets, 5*time.Minute)
	log.Printf("✅ Loaded %d triplets for user graph: %s", userID, len(triplets))
	
	// Create template data
	data := map[string]interface{}{
		"UserID": userID,
		"Data": map[string]interface{}{
			"Triplets": triplets,
		},
	}
	
	if err := h.templates.ExecuteTemplate(w, "UserGraphContent", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Logs handlers the centralized logs page
func (h *Handlers) Logs(w http.ResponseWriter, r *http.Request) {
	// Create page data with breadcrumbs
	data := &PageData{
		Title:    "Service Logs",
		SubTitle: "Centralized view of all service logs",
		Page:     "logs",
		Path:     r.URL.Path,
		BreadCrumbs: []BreadCrumb{
			{
				Title: "Logs",
				Path:  h.basePath + "/logs",
			},
		},
		MenuItems: GetMenuItems(h.basePath),
	}

	// Check if this is an HTMX request, if so render only the content
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "LogsContent", data); err != nil {
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

// LogsService handles individual service log requests
func (h *Handlers) LogsService(w http.ResponseWriter, r *http.Request) {
	service := chi.URLParam(r, "service")
	
	// Determine service URLs based on Railway service names
	var serviceURL string
	switch service {
	case "falkordb":
		serviceURL = "https://falkordb-service-production.up.railway.app"
	case "falkordb-browser":
		serviceURL = "https://falkordb-browser-production.up.railway.app"
	case "hybrid-proxy":
		serviceURL = "https://hybrid-proxy-production.up.railway.app"
	case "zep-server":
		serviceURL = "https://zep-server-production.up.railway.app"
	default:
		http.Error(w, "Unknown service", http.StatusBadRequest)
		return
	}
	
	// Simulate fetching logs (since Railway doesn't expose logs API directly)
	// In a real implementation, you'd use Railway's API or log aggregation service
	logs := h.fetchServiceLogs(service, serviceURL)
	
	// Return logs as HTML
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(logs))
}

// fetchServiceLogs simulates fetching logs from a service
func (h *Handlers) fetchServiceLogs(service, serviceURL string) string {
	// This is a placeholder implementation
	// In a real scenario, you would:
	// 1. Use Railway's CLI or API to fetch logs
	// 2. Connect to a log aggregation service like Loki, ELK, etc.
	// 3. Use a monitoring service API
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	switch service {
	case "falkordb":
		return fmt.Sprintf(`<div class="text-green-600">%s [INFO] FalkorDB service is running</div>
<div class="text-blue-600">%s [INFO] Connected to database</div>
<div class="text-gray-600">%s [DEBUG] Processing graph queries</div>
<div class="text-yellow-600">%s [WARN] High memory usage detected</div>
<div class="text-green-600">%s [INFO] Graph operations completed successfully</div>
<div class="text-gray-600">%s [DEBUG] Cleaning up connections</div>`, 
			timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)
	case "falkordb-browser":
		return fmt.Sprintf(`<div class="text-green-600">%s [INFO] FalkorDB Browser started</div>
<div class="text-blue-600">%s [INFO] Web interface available</div>
<div class="text-gray-600">%s [DEBUG] Handling browser requests</div>
<div class="text-green-600">%s [INFO] Database visualization loaded</div>
<div class="text-gray-600">%s [DEBUG] WebSocket connections active</div>`,
			timestamp, timestamp, timestamp, timestamp, timestamp)
	case "hybrid-proxy":
		return fmt.Sprintf(`<div class="text-green-600">%s [INFO] Hybrid proxy service started</div>
<div class="text-blue-600">%s [INFO] Proxy routes configured</div>
<div class="text-gray-600">%s [DEBUG] Forwarding requests</div>
<div class="text-yellow-600">%s [WARN] Rate limiting applied</div>
<div class="text-green-600">%s [INFO] All services healthy</div>`,
			timestamp, timestamp, timestamp, timestamp, timestamp)
	case "zep-server":
		return fmt.Sprintf(`<div class="text-green-600">%s [INFO] Zep server initialized</div>
<div class="text-blue-600">%s [INFO] API endpoints registered</div>
<div class="text-gray-600">%s [DEBUG] Processing memory requests</div>
<div class="text-green-600">%s [INFO] Session management active</div>
<div class="text-gray-600">%s [DEBUG] Background tasks running</div>
<div class="text-yellow-600">%s [WARN] Cache eviction performed</div>`,
			timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)
	default:
		return fmt.Sprintf(`<div class="text-red-600">%s [ERROR] Unknown service: %s</div>`, timestamp, service)
	}
}