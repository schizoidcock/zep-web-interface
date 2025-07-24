package zepapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Summary      map[string]interface{} `json:"summary,omitempty"`
	MessageCount int                    `json:"message_count,omitempty"`
}

type User struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email,omitempty"`
	FirstName    string    `json:"first_name,omitempty"`
	LastName     string    `json:"last_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SessionCount int       `json:"session_count,omitempty"`
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

	var sessions []Session
	if err := decodeResponse(resp, &sessions); err != nil {
		return nil, err
	}

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Try to decode as paginated response first
	var paginatedResp struct {
		Messages []Message `json:"messages"`
		Total    int       `json:"total"`
	}
	
	// Reset response body for reading
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		// If paginated response fails, try direct array
		resp, err = c.get(endpoint)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		
		var messages []Message
		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			return nil, 0, err
		}
		return messages, len(messages), nil
	}
	
	return paginatedResp.Messages, paginatedResp.Total, nil
}

func (c *Client) GetUsers() ([]User, error) {
	resp, err := c.get("/api/v2/users")
	if err != nil {
		return nil, err
	}

	var users []User
	if err := decodeResponse(resp, &users); err != nil {
		return nil, err
	}

	// TODO: Fetch session counts for each user from API if available
	// For now, set SessionCount to 0 to prevent template errors
	for i := range users {
		users[i].SessionCount = 0
	}

	return users, nil
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