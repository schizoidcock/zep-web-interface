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