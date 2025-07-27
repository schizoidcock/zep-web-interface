package zepapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, proxyURL string) *Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Configure proxy if provided
	if proxyURL != "" {
		if proxyParsed, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyParsed),
			}
		}
	}
	
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: client,
	}
}

func (c *Client) request(method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint
	
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		// Zep server expects "Api-Key" prefix, not "Bearer"
		req.Header.Set("Authorization", "Api-Key "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.request("GET", endpoint, nil)
}

func (c *Client) post(endpoint string, body interface{}) (*http.Response, error) {
	return c.request("POST", endpoint, body)
}

func (c *Client) delete(endpoint string) (*http.Response, error) {
	return c.request("DELETE", endpoint, nil)
}

func (c *Client) patch(endpoint string, body interface{}) (*http.Response, error) {
	return c.request("PATCH", endpoint, body)
}

// Helper function to decode JSON response
func decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	return json.NewDecoder(resp.Body).Decode(v)
}

// Data models for Zep v1.0.2 API responses
type Session struct {
	SessionID    string                 `json:"session_id"`
	UserID       string                 `json:"user_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	Summary      map[string]interface{} `json:"summary,omitempty"`
	MessageCount int                    `json:"message_count,omitempty"`
}

type User struct {
	UUID         string                 `json:"uuid"`
	ID           int64                  `json:"id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	DeletedAt    *time.Time             `json:"deleted_at,omitempty"`
	UserID       string                 `json:"user_id"`
	Email        string                 `json:"email,omitempty"`
	FirstName    string                 `json:"first_name,omitempty"`
	LastName     string                 `json:"last_name,omitempty"`
	ProjectUUID  string                 `json:"project_uuid"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SessionCount int                    `json:"session_count,omitempty"`
}

type Message struct {
	UUID       string                 `json:"uuid,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at,omitempty"`
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	TokenCount int                    `json:"token_count,omitempty"`
}

type SessionsResponse struct {
	Sessions []Session `json:"sessions"`
	Total    int       `json:"total"`
}

type UsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// API methods for Zep v1.0.2 (uses v2 API endpoints)
func (c *Client) GetSessions() ([]Session, error) {
	resp, err := c.get("/api/v2/sessions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Printf("🔍 DEBUG GetSessions - Status: %d, Response: %s", resp.StatusCode, string(body))
	
	// Also log the first session's project_uuid if available to help debug user filtering
	var tempSessions []map[string]interface{}
	if json.Unmarshal(body, &tempSessions) == nil && len(tempSessions) > 0 {
		if projectUUID, exists := tempSessions[0]["project_uuid"]; exists {
			log.Printf("🔍 DEBUG Sessions using project_uuid: %v", projectUUID)
		}
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var sessions []Session
	if err := json.Unmarshal(body, &sessions); err != nil {
		log.Printf("❌ Failed to unmarshal sessions: %v", err)
		return nil, err
	}

	log.Printf("✅ Parsed %d sessions", len(sessions))
	return sessions, nil
}

func (c *Client) GetSession(sessionID string) (*Session, error) {
	resp, err := c.get("/api/v2/sessions/" + sessionID)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := decodeResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (c *Client) DeleteSession(sessionID string) error {
	// Try different endpoints for session deletion based on v0.27 and v1.0.2 patterns
	endpoints := []string{
		"/api/v2/sessions/" + sessionID + "/memory", // Like v0.27 memory endpoint
		"/api/v2/sessions/" + sessionID,             // Direct session endpoint
	}
	
	var lastErr error
	for _, endpoint := range endpoints {
		resp, err := c.delete(endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		
		// If successful, return
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		
		// If 405 Method Not Allowed, try next endpoint
		if resp.StatusCode == http.StatusMethodNotAllowed {
			continue
		}
		
		// For other errors, return the error
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	// If all endpoints failed
	if lastErr != nil {
		return fmt.Errorf("failed to delete session, last error: %v", lastErr)
	}
	return fmt.Errorf("no working endpoint found for session deletion")
}

func (c *Client) GetMessageList(sessionID string, page, pageSize int) ([]Message, int, error) {
	// Build URL with pagination parameters
	endpoint := fmt.Sprintf("/api/v2/sessions/%s/messages?page=%d&page_size=%d", sessionID, page, pageSize)
	
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Printf("🔍 DEBUG GetMessageList - Session: %s, Status: %d, Response: %s", sessionID, resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Try to decode as paginated response first
	var paginatedResp struct {
		Messages []Message `json:"messages"`
		Total    int       `json:"total"`
	}
	
	if err := json.Unmarshal(body, &paginatedResp); err == nil && len(paginatedResp.Messages) > 0 {
		log.Printf("✅ Parsed %d messages from paginated response", len(paginatedResp.Messages))
		return paginatedResp.Messages, paginatedResp.Total, nil
	}

	// If paginated response fails, try direct array
	var messages []Message
	if err := json.Unmarshal(body, &messages); err != nil {
		log.Printf("❌ Failed to unmarshal messages: %v", err)
		return nil, 0, err
	}
	
	log.Printf("✅ Parsed %d messages from direct array", len(messages))
	return messages, len(messages), nil
}

func (c *Client) GetUsers() ([]User, error) {
	// Try the ordered users endpoint first, then fallback to simple endpoint
	endpoints := []string{
		"/api/v2/users-ordered?pageNumber=1&pageSize=100",
		"/api/v2/users?limit=100&cursor=0",
		"/api/v2/users",
	}
	
	var lastErr error
	for _, endpoint := range endpoints {
		resp, err := c.get(endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Read the raw response body for debugging
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}
		
		log.Printf("🔍 DEBUG GetUsers - Endpoint: %s, Status: %d, Response: %s", endpoint, resp.StatusCode, string(body))

		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			continue
		}

		// Try to parse as ordered response first
		var orderedResp struct {
			Users []User `json:"users"`
			Total int    `json:"total"`
		}
		if err := json.Unmarshal(body, &orderedResp); err == nil && len(orderedResp.Users) > 0 {
			log.Printf("✅ Parsed %d users from ordered response", len(orderedResp.Users))
			return orderedResp.Users, nil
		}

		// Try to parse as simple array
		var users []User
		if err := json.Unmarshal(body, &users); err == nil {
			log.Printf("✅ Parsed %d users from simple array", len(users))
			return users, nil
		}

		lastErr = fmt.Errorf("failed to parse response as users array or ordered response")
	}
	
	return nil, fmt.Errorf("all user endpoints failed, last error: %v", lastErr)
}

func (c *Client) GetUser(userID string) (*User, error) {
	resp, err := c.get("/api/v2/users/" + userID)
	if err != nil {
		return nil, err
	}

	var user User
	if err := decodeResponse(resp, &user); err != nil {
		return nil, err
	}

	// TODO: Fetch session count for this user from API if available
	user.SessionCount = 0

	return &user, nil
}

func (c *Client) GetUserSessions(userID string) ([]Session, error) {
	resp, err := c.get("/api/v2/users/" + userID + "/sessions")
	if err != nil {
		return nil, err
	}

	var sessions []Session
	if err := decodeResponse(resp, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// UpdateUser updates user information (firstName, lastName, email, metadata)
func (c *Client) UpdateUser(userID string, updateReq map[string]interface{}) (*User, error) {
	resp, err := c.patch("/api/v2/users/"+userID, updateReq)
	if err != nil {
		return nil, err
	}

	var user User
	if err := decodeResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(userID string) error {
	resp, err := c.delete("/api/v2/users/" + userID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteUserWithCleanup deletes a user and performs comprehensive cleanup
func (c *Client) DeleteUserWithCleanup(userID string) error {
	log.Printf("🧹 Starting comprehensive user deletion for: %s", userID)
	
	// Step 1: Get all sessions for this user first
	sessions, err := c.GetUserSessions(userID)
	if err != nil {
		log.Printf("⚠️ Could not get sessions for user %s (continuing): %v", userID, err)
		// Continue anyway, the sessions might not exist
	} else {
		log.Printf("📋 Found %d sessions for user %s", len(sessions), userID)
	}
	
	// Step 2: Delete each session individually (includes graph cleanup)
	for _, session := range sessions {
		log.Printf("🗑️ Deleting session: %s", session.SessionID)
		err := c.DeleteSession(session.SessionID)
		if err != nil {
			log.Printf("⚠️ Failed to delete session %s (continuing): %v", session.SessionID, err)
			// Continue with other sessions even if one fails
		} else {
			log.Printf("✅ Successfully deleted session: %s", session.SessionID)
		}
	}
	
	// Step 3: Try to cleanup graph data directly if we have a graph service
	// This is a safety net in case session deletion didn't clean everything
	if c.baseURL != "" {
		log.Printf("🧠 Attempting direct graph cleanup for user: %s", userID)
		err := c.DeleteUserGraphData(userID)
		if err != nil {
			log.Printf("⚠️ Direct graph cleanup failed (this is expected if no graph service): %v", err)
			// This is not a critical error - the graph service might not be available
		}
	}
	
	// Step 4: Delete the user from Zep server
	log.Printf("👤 Deleting user from Zep server: %s", userID)
	err = c.DeleteUser(userID)
	if err != nil {
		log.Printf("❌ Failed to delete user %s from Zep server: %v", userID, err)
		return fmt.Errorf("failed to delete user from Zep server: %w", err)
	}
	
	log.Printf("✅ Successfully completed comprehensive deletion for user: %s", userID)
	return nil
}

// DeleteUserGraphData attempts to cleanup graph data for a user (best effort)
func (c *Client) DeleteUserGraphData(userID string) error {
	// Try different graph service URLs that might be configured
	graphServiceURLs := []string{
		// Railway internal network patterns
		"http://zep-falkordb-service.railway.internal:8003",
		"http://zep-falkordb-service:8003",
		// Local development patterns
		"http://localhost:8003",
		"http://127.0.0.1:8003",
	}
	
	for _, baseURL := range graphServiceURLs {
		// Step 1: Clear all data from the user's database by calling group deletion
		groupDeleteURL := fmt.Sprintf("%s/group/%s", baseURL, userID)
		
		req, err := http.NewRequest("DELETE", groupDeleteURL, nil)
		if err != nil {
			continue
		}
		
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue // Try next URL
		}
		defer resp.Body.Close()
		
		// Check if group deletion succeeded
		if resp.StatusCode == 200 || resp.StatusCode == 404 {
			log.Printf("✅ Successfully cleared graph data for user %s via %s", userID, baseURL)
			
			// Step 2: Delete the entire database (should now be empty)
			databaseDeleteURL := fmt.Sprintf("%s/database/%s", baseURL, userID)
			
			dbReq, err := http.NewRequest("DELETE", databaseDeleteURL, nil)
			if err != nil {
				log.Printf("⚠️ Failed to create database deletion request for user %s: %v", userID, err)
				return nil // Data cleanup succeeded, database deletion is secondary
			}
			
			dbReq.Header.Set("Content-Type", "application/json")
			
			dbResp, err := client.Do(dbReq)
			if err != nil {
				log.Printf("⚠️ Database deletion request failed for user %s: %v", userID, err)
				return nil // Data cleanup succeeded, database deletion is secondary
			}
			defer dbResp.Body.Close()
			
			if dbResp.StatusCode == 200 || dbResp.StatusCode == 404 {
				log.Printf("✅ Successfully deleted database for user %s via %s", userID, baseURL)
			} else {
				dbBody, _ := io.ReadAll(dbResp.Body)
				log.Printf("⚠️ Database deletion returned %d for user %s: %s", dbResp.StatusCode, userID, string(dbBody))
			}
			
			return nil
		}
		
		// Log what we got but continue trying
		body, _ := io.ReadAll(resp.Body)
		log.Printf("🔍 Graph cleanup attempt via %s returned %d: %s", baseURL, resp.StatusCode, string(body))
	}
	
	return fmt.Errorf("no graph service responded successfully (tried %d URLs)", len(graphServiceURLs))
}

// CreateUser creates a new user
func (c *Client) CreateUser(createReq map[string]interface{}) (*User, error) {
	resp, err := c.post("/api/v2/users", createReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

// GetSystemStats retrieves system statistics for the settings page
func (c *Client) GetSystemStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get total users count
	if users, err := c.GetUsers(); err == nil {
		stats["total_users"] = len(users)
	} else {
		stats["total_users"] = 0
		log.Printf("❌ Failed to get users count: %v", err)
	}
	
	// Get total sessions count
	if sessions, err := c.GetSessions(); err == nil {
		stats["total_sessions"] = len(sessions)
		
		// Calculate additional session statistics
		activeCount := 0
		for _, session := range sessions {
			if session.EndedAt == nil {
				activeCount++
			}
		}
		stats["active_sessions"] = activeCount
		stats["ended_sessions"] = len(sessions) - activeCount
	} else {
		stats["total_sessions"] = 0
		stats["active_sessions"] = 0
		stats["ended_sessions"] = 0
		log.Printf("❌ Failed to get sessions count: %v", err)
	}
	
	return stats, nil
}

// GetServerHealth checks server health and version
func (c *Client) GetServerHealth() (map[string]interface{}, error) {
	resp, err := c.get("/healthz")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	health := make(map[string]interface{})
	health["status"] = "healthy"
	health["status_code"] = resp.StatusCode
	
	// Extract version from headers if available
	if version := resp.Header.Get("X-Zep-Version"); version != "" {
		health["version"] = version
	} else {
		health["version"] = "unknown"
	}
	
	// Record response time
	health["response_time"] = "< 1ms" // Placeholder since we don't have timing here
	
	return health, nil
}