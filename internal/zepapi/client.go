package zepapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, proxyURL string) *Client {
	// Create optimized transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	
	// Configure proxy if provided
	if proxyURL != "" {
		if proxyParsed, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyParsed)
		}
	}
	
	client := &http.Client{
		Timeout:   10 * time.Second, // Reduced from 30s for better UX
		Transport: transport,
	}
	
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: client,
	}
}

func (c *Client) request(method, endpoint string, body interface{}) (*http.Response, error) {
	return c.requestWithContext(context.Background(), method, endpoint, body)
}

func (c *Client) requestWithContext(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint
	
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "zep-web-interface/1.0")
	if c.apiKey != "" {
		// Use Bearer format for zep-server-railway Go REST API
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

type Episode struct {
	EpisodeID   string     `json:"uuid"`
	Content     string     `json:"content"`
	Source      string     `json:"source"`
	Description string     `json:"source_description,omitempty"`
	Status      string     `json:"status,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	Role        string     `json:"role,omitempty"`
	Processed   bool       `json:"processed,omitempty"`
}

// Graph data structures - Episodes are used as edges in Zep's graph implementation
type GraphNode struct {
	UUID       string                 `json:"uuid"`
	Name       string                 `json:"name"`
	Summary    string                 `json:"summary,omitempty"`
	Labels     []string               `json:"labels,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// GraphEpisode represents the relationship/edge between nodes in Zep's graph
type GraphEpisode struct {
	UUID             string    `json:"uuid"`
	SourceNodeUUID   string    `json:"source_node_uuid"`
	TargetNodeUUID   string    `json:"target_node_uuid"`
	Type             string    `json:"type"`
	Name             string    `json:"name"`
	Fact             string    `json:"fact,omitempty"`
	Content          string    `json:"content,omitempty"`
	Summary          string    `json:"summary,omitempty"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	ValidAt          string    `json:"valid_at,omitempty"`
	ExpiredAt        string    `json:"expired_at,omitempty"`
	InvalidAt        string    `json:"invalid_at,omitempty"`
}

// RawTriplet represents a graph triplet with episode as the connecting relationship
type RawTriplet struct {
	SourceNode GraphNode    `json:"sourceNode"`
	Episode    GraphEpisode `json:"episode"`
	TargetNode GraphNode    `json:"targetNode"`
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
	resp, err := c.get("/api/v2/sessions-ordered")
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as paginated response first
	var paginatedResp struct {
		Sessions []Session `json:"sessions"`
		Total    int       `json:"total_count"`
		TotalCount int     `json:"total_count"`
		RowCount   int     `json:"row_count"`
	}
	
	if err := json.Unmarshal(body, &paginatedResp); err == nil {
		if len(paginatedResp.Sessions) > 0 {
			log.Printf("✅ Parsed %d sessions from paginated response", len(paginatedResp.Sessions))
			return paginatedResp.Sessions, nil
		}
	}

	// Try to decode as object with nested sessions array
	var responseObj map[string]interface{}
	if err := json.Unmarshal(body, &responseObj); err == nil {
		if sessions, ok := responseObj["sessions"].([]interface{}); ok {
			parsedSessions := make([]Session, 0, len(sessions))
			for _, session := range sessions {
				if sessionMap, ok := session.(map[string]interface{}); ok {
					sess := Session{
						SessionID: getStringFromInterface(sessionMap["session_id"]),
						UserID:    getStringFromInterface(sessionMap["user_id"]),
					}
					
					// Parse timestamps
					if createdAtStr := getStringFromInterface(sessionMap["created_at"]); createdAtStr != "" {
						if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
							sess.CreatedAt = createdAt
						}
					}
					if updatedAtStr := getStringFromInterface(sessionMap["updated_at"]); updatedAtStr != "" {
						if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
							sess.UpdatedAt = updatedAt
						}
					}
					
					// Handle message count
					if msgCount, ok := sessionMap["message_count"].(float64); ok {
						sess.MessageCount = int(msgCount)
					}
					
					parsedSessions = append(parsedSessions, sess)
				}
			}
			
			log.Printf("✅ Parsed %d sessions from object response", len(parsedSessions))
			return parsedSessions, nil
		}
	}

	// Fallback to direct array
	var sessions []Session
	if err := json.Unmarshal(body, &sessions); err == nil {
		log.Printf("✅ Parsed %d sessions from direct array", len(sessions))
		return sessions, nil
	}
	
	// If all parsing attempts fail, return empty slice instead of error
	log.Printf("⚠️ No sessions found or unknown format, returning empty slice")
	return []Session{}, nil
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
	// Use the official Zep API endpoint for session memory deletion
	resp, err := c.delete("/api/v2/sessions/" + sessionID + "/memory")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Check for successful deletion
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	
	// Handle error response
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
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

	// Try to decode as paginated response first (multiple possible formats)
	var paginatedResp struct {
		Messages []Message `json:"messages"`
		Total    int       `json:"total"`
		TotalCount int     `json:"total_count"`
	}
	
	if err := json.Unmarshal(body, &paginatedResp); err == nil {
		if len(paginatedResp.Messages) > 0 {
			total := paginatedResp.Total
			if total == 0 {
				total = paginatedResp.TotalCount
			}
			log.Printf("✅ Parsed %d messages from paginated response", len(paginatedResp.Messages))
			return paginatedResp.Messages, total, nil
		}
	}

	// Try to decode as object with nested messages array
	var responseObj map[string]interface{}
	if err := json.Unmarshal(body, &responseObj); err == nil {
		// Check for messages key
		if msgs, ok := responseObj["messages"].([]interface{}); ok {
			messages := make([]Message, 0, len(msgs))
			for _, msg := range msgs {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					message := Message{
						UUID:      getStringFromInterface(msgMap["uuid"]),
						Role:      getStringFromInterface(msgMap["role"]),
						Content:   getStringFromInterface(msgMap["content"]),
						Metadata:  getMapFromInterface(msgMap["metadata"]),
					}
					
					// Parse timestamps
					if createdAtStr := getStringFromInterface(msgMap["created_at"]); createdAtStr != "" {
						if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
							message.CreatedAt = createdAt
						}
					}
					
					// Handle token count
					if tokenCount, ok := msgMap["token_count"].(float64); ok {
						message.TokenCount = int(tokenCount)
					}
					
					messages = append(messages, message)
				}
			}
			
			total := len(messages)
			if totalCount, ok := responseObj["total_count"].(float64); ok {
				total = int(totalCount)
			} else if totalCount, ok := responseObj["total"].(float64); ok {
				total = int(totalCount)
			}
			
			log.Printf("✅ Parsed %d messages from object response", len(messages))
			return messages, total, nil
		}
	}

	// Try direct array as fallback
	var messages []Message
	if err := json.Unmarshal(body, &messages); err == nil {
		log.Printf("✅ Parsed %d messages from direct array", len(messages))
		return messages, len(messages), nil
	}
	
	// If all parsing attempts fail, return empty slice instead of error
	log.Printf("⚠️ No messages found or unknown format, returning empty slice")
	return []Message{}, 0, nil
}

func (c *Client) GetUsers() ([]User, error) {
	// Use the proper ordered users endpoint as per official API
	resp, err := c.get("/api/v2/users-ordered?pageNumber=1&pageSize=100")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Printf("🔍 DEBUG GetUsers - Status: %d, Response: %s", resp.StatusCode, string(body))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as ordered response first
	var orderedResp struct {
		Users     []User `json:"users"`
		RowCount  int    `json:"row_count"`
		TotalCount int   `json:"total_count"`
		Total     int    `json:"total"`
	}
	if err := json.Unmarshal(body, &orderedResp); err == nil {
		if len(orderedResp.Users) > 0 {
			log.Printf("✅ Parsed %d users from ordered response", len(orderedResp.Users))
			return orderedResp.Users, nil
		}
	}

	// Try to decode as object with nested users array
	var responseObj map[string]interface{}
	if err := json.Unmarshal(body, &responseObj); err == nil {
		if users, ok := responseObj["users"].([]interface{}); ok {
			parsedUsers := make([]User, 0, len(users))
			for _, user := range users {
				if userMap, ok := user.(map[string]interface{}); ok {
					usr := User{
						UUID:    getStringFromInterface(userMap["uuid"]),
						UserID:  getStringFromInterface(userMap["user_id"]),
						Email:   getStringFromInterface(userMap["email"]),
						FirstName: getStringFromInterface(userMap["first_name"]),
						LastName:  getStringFromInterface(userMap["last_name"]),
						ProjectUUID: getStringFromInterface(userMap["project_uuid"]),
						Metadata: getMapFromInterface(userMap["metadata"]),
					}
					
					// Parse timestamps
					if createdAtStr := getStringFromInterface(userMap["created_at"]); createdAtStr != "" {
						if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
							usr.CreatedAt = createdAt
						}
					}
					if updatedAtStr := getStringFromInterface(userMap["updated_at"]); updatedAtStr != "" {
						if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
							usr.UpdatedAt = updatedAt
						}
					}
					
					// Handle session count
					if sessionCount, ok := userMap["session_count"].(float64); ok {
						usr.SessionCount = int(sessionCount)
					}
					
					parsedUsers = append(parsedUsers, usr)
				}
			}
			
			log.Printf("✅ Parsed %d users from object response", len(parsedUsers))
			return parsedUsers, nil
		}
	}

	// Fallback to direct array
	var users []User
	if err := json.Unmarshal(body, &users); err == nil {
		log.Printf("✅ Parsed %d users from direct array", len(users))
		return users, nil
	}
	
	// If all parsing attempts fail, return empty slice instead of error
	log.Printf("⚠️ No users found or unknown format, returning empty slice")
	return []User{}, nil
}

// GetUsersWithSessionCounts fetches users with their session counts in a single optimized call
func (c *Client) GetUsersWithSessionCounts() ([]User, error) {
	users, err := c.GetUsers()
	if err != nil {
		return nil, err
	}

	// Create concurrent channel-based session count fetcher
	type sessionCountResult struct {
		index int
		count int
		err   error
	}

	resultChan := make(chan sessionCountResult, len(users))
	
	// Limit concurrent requests to avoid overwhelming the server
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	// Fetch session counts concurrently
	for i := range users {
		go func(idx int, userID string) {
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			sessions, err := c.GetUserSessions(userID)
			if err != nil {
				resultChan <- sessionCountResult{index: idx, count: 0, err: err}
				return
			}
			resultChan <- sessionCountResult{index: idx, count: len(sessions), err: nil}
		}(i, users[i].UserID)
	}

	// Collect results
	for i := 0; i < len(users); i++ {
		result := <-resultChan
		if result.err != nil {
			log.Printf("⚠️ Failed to get session count for user %s: %v", users[result.index].UserID, result.err)
			users[result.index].SessionCount = 0
		} else {
			users[result.index].SessionCount = result.count
		}
	}

	log.Printf("✅ Fetched session counts for %d users concurrently", len(users))
	return users, nil
}

func (c *Client) GetUsersLegacy() ([]User, error) {
	// Fallback method with multiple endpoints for compatibility
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

// Helper function to safely get string value from pointer
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Helper function to safely get string from interface value
func getStringFromInterface(val interface{}) string {
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}

// Helper function to safely get map from interface value
func getMapFromInterface(val interface{}) map[string]interface{} {
	if val == nil {
		return nil
	}
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetUserGraphTriplets fetches graph triplets for a specific user with optimized concurrent processing
func (c *Client) GetUserGraphTriplets(userID string) ([]RawTriplet, error) {
	// Step 1: Get user episodes first
	episodes, err := c.GetUserEpisodes(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user episodes: %w", err)
	}

	log.Printf("🔍 Found %d episodes for user %s", len(episodes), userID)

	if len(episodes) == 0 {
		return []RawTriplet{}, nil
	}

	var triplets []RawTriplet
	var mu sync.Mutex
	nodeMap := make(map[string]GraphNode) // Track unique nodes
	
	// Use worker pool for concurrent episode processing
	const maxWorkers = 3 // Limit concurrent requests
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	
	// Process episodes concurrently
	for _, episode := range episodes {
		wg.Add(1)
		go func(ep Episode) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			mentions, err := c.GetEpisodeMentions(ep.EpisodeID)
			if err != nil {
				log.Printf("⚠️ Failed to get mentions for episode %s: %v", ep.EpisodeID, err)
				return
			}

			mu.Lock()
			defer mu.Unlock()

			// Store all nodes from this episode
			for _, node := range mentions.Nodes {
				nodeMap[node.UUID] = GraphNode{
					UUID:       node.UUID,
					Name:       node.Name,
					Summary:    node.Summary,
					Labels:     node.Labels,
					Attributes: node.Attributes,
					CreatedAt:  node.CreatedAt,
					UpdatedAt:  node.UpdatedAt,
				}
			}

			// Build triplets from edges
			for _, edge := range mentions.Edges {
				sourceNode, sourceExists := nodeMap[edge.SourceNodeUUID]
				targetNode, targetExists := nodeMap[edge.TargetNodeUUID]
				
				if !sourceExists || !targetExists {
					// Check if nodes are in current batch
					for _, node := range mentions.Nodes {
						if node.UUID == edge.SourceNodeUUID {
							sourceNode = GraphNode{
								UUID: node.UUID, Name: node.Name, Summary: node.Summary,
								Labels: node.Labels, Attributes: node.Attributes,
								CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt,
							}
							sourceExists = true
						}
						if node.UUID == edge.TargetNodeUUID {
							targetNode = GraphNode{
								UUID: node.UUID, Name: node.Name, Summary: node.Summary,
								Labels: node.Labels, Attributes: node.Attributes,
								CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt,
							}
							targetExists = true
						}
					}
				}
				
				if !sourceExists || !targetExists {
					log.Printf("⚠️ Missing nodes for edge %s (source: %v, target: %v)", edge.UUID, sourceExists, targetExists)
					continue
				}

				triplet := RawTriplet{
					SourceNode: sourceNode,
					Episode: GraphEpisode{
						UUID:           edge.UUID,
						SourceNodeUUID: edge.SourceNodeUUID,
						TargetNodeUUID: edge.TargetNodeUUID,
						Type:           "relationship",
						Name:           edge.Name,
						Fact:           edge.Fact,
						Content:        ep.Content,
						Summary:        ep.Description,
						CreatedAt:      edge.CreatedAt,
						UpdatedAt:      edge.UpdatedAt,
						ValidAt:        getStringValue(edge.ValidAt),
						ExpiredAt:      getStringValue(edge.ExpiredAt),
						InvalidAt:      getStringValue(edge.InvalidAt),
					},
					TargetNode: targetNode,
				}
				
				triplets = append(triplets, triplet)
			}
		}(episode)
	}

	wg.Wait()

	log.Printf("✅ Built %d graph triplets for user %s from %d episodes (concurrent)", len(triplets), userID, len(episodes))
	return triplets, nil
}

// GetEpisodeMentions fetches nodes and edges mentioned in a specific episode
func (c *Client) GetEpisodeMentions(episodeUUID string) (*EpisodeMentions, error) {
	resp, err := c.get("/api/v2/graph/episodes/" + episodeUUID + "/mentions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("episode mentions API error %d: %s", resp.StatusCode, string(body))
	}

	var mentions EpisodeMentions
	if err := json.Unmarshal(body, &mentions); err != nil {
		log.Printf("❌ Failed to unmarshal episode mentions: %v", err)
		return nil, err
	}

	log.Printf("✅ Got %d nodes and %d edges for episode %s", len(mentions.Nodes), len(mentions.Edges), episodeUUID)
	return &mentions, nil
}

// EpisodeMentions represents nodes and edges mentioned in an episode
type EpisodeMentions struct {
	Nodes []*EntityNode `json:"nodes,omitempty"`
	Edges []*EntityEdge `json:"edges,omitempty"`
}

// EntityNode represents a node in the graph
type EntityNode struct {
	UUID       string                 `json:"uuid"`
	Name       string                 `json:"name"`
	Summary    string                 `json:"summary"`
	Labels     []string               `json:"labels,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Score      *float64               `json:"score,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// EntityEdge represents an edge in the graph
type EntityEdge struct {
	UUID           string                 `json:"uuid"`
	SourceNodeUUID string                 `json:"source_node_uuid"`
	TargetNodeUUID string                 `json:"target_node_uuid"`
	Name           string                 `json:"name"`
	Fact           string                 `json:"fact"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	Episodes       []string               `json:"episodes,omitempty"`
	Score          *float64               `json:"score,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	ValidAt        *string                `json:"valid_at,omitempty"`
	ExpiredAt      *string                `json:"expired_at,omitempty"`
	InvalidAt      *string                `json:"invalid_at,omitempty"`
}

// GetUserEpisodes fetches episodes for a specific user from the graph API
func (c *Client) GetUserEpisodes(userID string) ([]Episode, error) {
	resp, err := c.get("/api/v2/graph/episodes/user/" + userID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Printf("🔍 DEBUG GetUserEpisodes - User: %s, Status: %d, Response: %s", userID, resp.StatusCode, string(body))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var episodes []Episode
	if err := json.Unmarshal(body, &episodes); err != nil {
		log.Printf("❌ Failed to unmarshal episodes: %v", err)
		return nil, err
	}

	log.Printf("✅ Parsed %d episodes for user %s", len(episodes), userID)
	return episodes, nil
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

// DeleteUserWithCleanup deletes a user and performs comprehensive cleanup with optimized concurrency
func (c *Client) DeleteUserWithCleanup(userID string) error {
	log.Printf("🧹 Starting optimized user deletion for: %s", userID)
	
	// Step 1: Get all sessions for this user first
	sessions, err := c.GetUserSessions(userID)
	if err != nil {
		log.Printf("⚠️ Could not get sessions for user %s (continuing): %v", userID, err)
		sessions = []Session{} // Continue with empty sessions
	} else {
		log.Printf("📋 Found %d sessions for user %s", len(sessions), userID)
	}
	
	// Step 2: Delete sessions concurrently (major optimization)
	if len(sessions) > 0 {
		c.deleteSessionsConcurrently(sessions)
	}
	
	// Step 3: Start graph cleanup in background (non-blocking)
	go func() {
		log.Printf("🧠 Starting background graph cleanup for user: %s", userID)
		err := c.DeleteUserGraphData(userID)
		if err != nil {
			log.Printf("⚠️ Background graph cleanup failed for %s: %v", userID, err)
		} else {
			log.Printf("✅ Background graph cleanup completed for %s", userID)
		}
	}()
	
	// Step 4: Delete the user from Zep server (most critical step)
	log.Printf("👤 Deleting user from Zep server: %s", userID)
	err = c.DeleteUser(userID)
	if err != nil {
		log.Printf("❌ Failed to delete user %s from Zep server: %v", userID, err)
		return fmt.Errorf("failed to delete user from Zep server: %w", err)
	}
	
	log.Printf("✅ User deletion completed for: %s (graph cleanup continues in background)", userID)
	return nil
}

// BulkDeleteUsers deletes multiple users concurrently with progress tracking
func (c *Client) BulkDeleteUsers(userIDs []string, progressCallback func(completed, total int, userID string, err error)) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("no users provided for bulk deletion")
	}
	
	log.Printf("🚀 Starting bulk deletion of %d users", len(userIDs))
	
	// Limit concurrent deletions to avoid overwhelming the server
	maxWorkers := 2 // Conservative limit for user deletions
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var completed int
	var mu sync.Mutex
	
	for _, userID := range userIDs {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release
			
			err := c.DeleteUserWithCleanup(uid)
			
			mu.Lock()
			completed++
			currentCompleted := completed
			mu.Unlock()
			
			if progressCallback != nil {
				progressCallback(currentCompleted, len(userIDs), uid, err)
			}
			
			if err != nil {
				log.Printf("❌ Bulk deletion failed for user %s: %v", uid, err)
			} else {
				log.Printf("✅ Bulk deletion completed for user %s (%d/%d)", uid, currentCompleted, len(userIDs))
			}
		}(userID)
	}
	
	wg.Wait()
	log.Printf("✅ Bulk deletion completed for all %d users", len(userIDs))
	return nil
}

// deleteSessionsConcurrently deletes multiple sessions in parallel
func (c *Client) deleteSessionsConcurrently(sessions []Session) {
	log.Printf("🚀 Starting concurrent session deletion for %d sessions", len(sessions))
	
	// Limit concurrent deletions to avoid overwhelming the server
	maxWorkers := 3
	if len(sessions) < maxWorkers {
		maxWorkers = len(sessions)
	}
	
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	
	for _, session := range sessions {
		wg.Add(1)
		go func(s Session) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release
			
			log.Printf("🗑️ Deleting session: %s", s.SessionID)
			err := c.DeleteSession(s.SessionID)
			if err != nil {
				log.Printf("⚠️ Failed to delete session %s: %v", s.SessionID, err)
			} else {
				log.Printf("✅ Successfully deleted session: %s", s.SessionID)
			}
		}(session)
	}
	
	wg.Wait()
	log.Printf("✅ Concurrent session deletion completed")
}

// DeleteUserGraphData attempts to cleanup graph data for a user with optimized timeouts
func (c *Client) DeleteUserGraphData(userID string) error {
	// Try different graph service URLs with reduced timeouts
	graphServiceURLs := []string{
		// Railway internal network patterns (most likely to work)
		"http://graphiti-service.railway.internal:8000",
		"http://graphiti-service:8000",
		// Local development patterns
		"http://localhost:8000",
		"http://127.0.0.1:8000",
	}
	
	// Use shorter timeout for faster failure detection
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, baseURL := range graphServiceURLs {
		// Try group deletion with timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		groupDeleteURL := fmt.Sprintf("%s/group/%s", baseURL, userID)
		req, err := http.NewRequestWithContext(ctx, "DELETE", groupDeleteURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("🔍 Graph service %s not reachable: %v", baseURL, err)
			continue // Try next URL quickly
		}
		defer resp.Body.Close()
		
		// Check if group deletion succeeded
		if resp.StatusCode == 200 || resp.StatusCode == 404 {
			log.Printf("✅ Successfully cleared graph data for user %s via %s", userID, baseURL)
			
			// Step 2: Delete the entire database (optional, background)
			go func(dbURL string) {
				databaseDeleteURL := fmt.Sprintf("%s/database/%s", dbURL, userID)
				dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer dbCancel()
				
				dbReq, err := http.NewRequestWithContext(dbCtx, "DELETE", databaseDeleteURL, nil)
				if err != nil {
					log.Printf("⚠️ Failed to create database deletion request: %v", err)
					return
				}
				dbReq.Header.Set("Content-Type", "application/json")
				
				dbResp, err := client.Do(dbReq)
				if err != nil {
					log.Printf("⚠️ Database deletion failed: %v", err)
					return
				}
				defer dbResp.Body.Close()
				
				if dbResp.StatusCode == 200 || dbResp.StatusCode == 404 {
					log.Printf("✅ Database deletion completed for user %s", userID)
				} else {
					log.Printf("⚠️ Database deletion returned %d for user %s", dbResp.StatusCode, userID)
				}
			}(baseURL)
			
			return nil // Success - don't try other URLs
		}
		
		// Log failed attempt but continue quickly
		log.Printf("🔍 Graph service %s returned %d, trying next", baseURL, resp.StatusCode)
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

// GetServerHealth checks server health and returns actual status
func (c *Client) GetServerHealth() (map[string]interface{}, error) {
	fullURL := strings.TrimRight(c.baseURL, "/") + "/health"
	log.Printf("🔍 DEBUG Checking health at: %s (v2)", fullURL)
	
	resp, err := c.get("/health")
	if err != nil {
		log.Printf("❌ Health check failed: %v", err)
		return map[string]interface{}{
			"status":  "unhealthy",
			"version": "unknown",
			"error":   err.Error(),
		}, nil
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	log.Printf("🔍 DEBUG Health response - Status: %d, Body: %s", resp.StatusCode, string(body))
	
	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		log.Printf("⚠️ Failed to parse health JSON: %v", err)
		// Fallback based on HTTP status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			responseData = map[string]interface{}{
				"status":  "healthy",
				"version": "unknown",
			}
		} else {
			responseData = map[string]interface{}{
				"status":  "unhealthy",
				"version": "unknown",
			}
		}
	}
	
	// Ensure we have a status field
	if status, ok := responseData["status"].(string); ok {
		responseData["status"] = strings.ToLower(status)
	} else {
		// Default to healthy if no status field but 200 response
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			responseData["status"] = "healthy"
		} else {
			responseData["status"] = "unhealthy"
		}
	}
	
	// Ensure version exists
	if _, ok := responseData["version"]; !ok {
		responseData["version"] = "unknown"
	}
	
	log.Printf("✅ Health check result: %+v", responseData)
	return responseData, nil
}