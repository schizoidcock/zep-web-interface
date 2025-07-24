package handlers

import (
	"html/template"
	"net/http"
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
	for i, session := range sessions {
		sessionRows[i] = SessionRow{Session: &session}
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
	
	session, err := h.apiClient.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create page data with breadcrumbs like other handlers
	pageData := &PageData{
		Title:    "Session Details",
		SubTitle: "View session information and chat history",
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
			// Store session in a custom field
		},
		MenuItems: MenuItems,
	}

	// Add session and empty messages data for template access
	data := map[string]interface{}{
		"Title":      pageData.Title,
		"SubTitle":   pageData.SubTitle,
		"Page":       pageData.Page,
		"Path":       pageData.Path,
		"BreadCrumbs": pageData.BreadCrumbs,
		"MenuItems":  pageData.MenuItems,
		"Data": map[string]interface{}{
			"Session": session,
			"Messages": []interface{}{}, // Empty messages array to prevent ChatHistory errors
			"TableID": "chat-history",
			"TotalCount": 0,
			"CurrentPage": 1,
			"PageCount": 1,
			"PageSize": 10,
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

// UserList handles the users list page
func (h *Handlers) UserList(w http.ResponseWriter, r *http.Request) {
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
	
	user, err := h.apiClient.GetUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create page data with breadcrumbs and proper data structure
	data := map[string]interface{}{
		"Title": "User Details",
		"SubTitle": "View and manage user information",
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
		"Data": user, // User data directly as Data for the template to access .Data.FirstName, etc.
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
	for i, session := range sessions {
		sessionRows[i] = SessionRow{Session: &session}
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

// Settings handles the settings page
func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	// Create mock config HTML for now - TODO: fetch from Zep API if available
	configHTML := `
	<div class="mb-4">
		<h3 class="text-lg font-semibold text-gray-800">Zep Server Configuration</h3>
		<p class="text-gray-600 mt-2">Configuration details would appear here when available from the API.</p>
	</div>
	<div class="bg-gray-50 p-4 rounded border">
		<code class="text-sm">
		API URL: Connected<br/>
		Authentication: Active<br/>
		Version: v1.0.2<br/>
		</code>
	</div>
	`

	// Create page data with breadcrumbs
	pageData := &PageData{
		Title:    "Settings",
		SubTitle: "Server configuration and settings",
		Page:     "settings",
		Path:     r.URL.Path,
		BreadCrumbs: []BreadCrumb{
			{
				Title: "Settings",
				Path:  "/settings",
			},
		},
		Data: &TableData{
			// Use a custom field for ConfigHTML
		},
		MenuItems: MenuItems,
	}
	
	// Add ConfigHTML to the data map
	data := map[string]interface{}{
		"Title":      pageData.Title,
		"SubTitle":   pageData.SubTitle,
		"Page":       pageData.Page,
		"Path":       pageData.Path,
		"BreadCrumbs": pageData.BreadCrumbs,
		"MenuItems":  pageData.MenuItems,
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
	for i, session := range sessions {
		sessionRows[i] = SessionRow{Session: &session}
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