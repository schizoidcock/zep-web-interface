package zepapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.request("GET", endpoint, nil)
}

func (c *Client) post(endpoint string, body interface{}) (*http.Response, error) {
	return c.request("POST", endpoint, body)
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
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	MessageCount int                   `json:"message_count,omitempty"`
}

type User struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SessionsResponse struct {
	Sessions []Session `json:"sessions"`
	Total    int       `json:"total"`
}

type UsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// API methods for Zep v1.0.2
func (c *Client) GetSessions() ([]Session, error) {
	resp, err := c.get("/api/v1/sessions")
	if err != nil {
		return nil, err
	}

	var sessionsResp SessionsResponse
	if err := decodeResponse(resp, &sessionsResp); err != nil {
		return nil, err
	}

	return sessionsResp.Sessions, nil
}

func (c *Client) GetSession(sessionID string) (*Session, error) {
	resp, err := c.get("/api/v1/sessions/" + sessionID)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := decodeResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (c *Client) GetUsers() ([]User, error) {
	resp, err := c.get("/api/v1/users")
	if err != nil {
		return nil, err
	}

	var usersResp UsersResponse
	if err := decodeResponse(resp, &usersResp); err != nil {
		return nil, err
	}

	return usersResp.Users, nil
}

func (c *Client) GetUser(userID string) (*User, error) {
	resp, err := c.get("/api/v1/users/" + userID)
	if err != nil {
		return nil, err
	}

	var user User
	if err := decodeResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) GetUserSessions(userID string) ([]Session, error) {
	resp, err := c.get("/api/v1/users/" + userID + "/sessions")
	if err != nil {
		return nil, err
	}

	var sessionsResp SessionsResponse
	if err := decodeResponse(resp, &sessionsResp); err != nil {
		return nil, err
	}

	return sessionsResp.Sessions, nil
}