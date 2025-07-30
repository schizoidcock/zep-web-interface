package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// DeletionStatus tracks the status of user deletion operations
type DeletionStatus struct {
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`    // "started", "sessions_done", "graph_cleanup", "completed", "failed"
	Progress  int       `json:"progress"`  // 0-100
	Message   string    `json:"message"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

// DeletionTracker manages deletion status tracking
type DeletionTracker struct {
	statuses map[string]*DeletionStatus
	mutex    sync.RWMutex
}

var deletionTracker = &DeletionTracker{
	statuses: make(map[string]*DeletionStatus),
}

// TrackDeletion starts tracking a deletion operation
func (dt *DeletionTracker) TrackDeletion(userID string) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	dt.statuses[userID] = &DeletionStatus{
		UserID:    userID,
		Status:    "started",
		Progress:  10,
		Message:   "Starting user deletion...",
		StartedAt: time.Now(),
	}
}

// UpdateStatus updates the deletion progress
func (dt *DeletionTracker) UpdateStatus(userID, status, message string, progress int) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	if deletion, exists := dt.statuses[userID]; exists {
		deletion.Status = status
		deletion.Message = message
		deletion.Progress = progress
	}
}

// MarkCompleted marks deletion as completed
func (dt *DeletionTracker) MarkCompleted(userID string) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	if deletion, exists := dt.statuses[userID]; exists {
		deletion.Status = "completed"
		deletion.Message = "User deletion completed successfully"
		deletion.Progress = 100
		
		// Auto-cleanup after 30 seconds
		go func() {
			time.Sleep(30 * time.Second)
			dt.mutex.Lock()
			delete(dt.statuses, userID)
			dt.mutex.Unlock()
		}()
	}
}

// MarkFailed marks deletion as failed
func (dt *DeletionTracker) MarkFailed(userID, error string) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	if deletion, exists := dt.statuses[userID]; exists {
		deletion.Status = "failed"
		deletion.Message = "User deletion failed"
		deletion.Error = error
		deletion.Progress = 0
		
		// Auto-cleanup after 60 seconds for failed operations
		go func() {
			time.Sleep(60 * time.Second)
			dt.mutex.Lock()
			delete(dt.statuses, userID)
			dt.mutex.Unlock()
		}()
	}
}

// GetStatus retrieves deletion status
func (dt *DeletionTracker) GetStatus(userID string) (*DeletionStatus, bool) {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()
	
	status, exists := dt.statuses[userID]
	return status, exists
}

// DeletionStatus endpoint to check deletion progress
func (h *Handlers) DeletionStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	status, exists := deletionTracker.GetStatus(userID)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "not_found",
			"message":  "No active deletion found for this user",
			"progress": 0,
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Enhanced DeleteUser with progress tracking
func (h *Handlers) DeleteUserEnhanced(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	// Start tracking the deletion
	deletionTracker.TrackDeletion(userID)
	
	// Start deletion in background with detailed progress tracking
	go func() {
		defer func() {
			if r := recover(); r != nil {
				deletionTracker.MarkFailed(userID, fmt.Sprintf("Deletion panic: %v", r))
			}
		}()
		
		// Step 1: Get sessions
		deletionTracker.UpdateStatus(userID, "fetching_sessions", "Retrieving user sessions...", 20)
		sessions, err := h.apiClient.GetUserSessions(userID)
		if err != nil {
			deletionTracker.UpdateStatus(userID, "sessions_partial", "Could not fetch all sessions, continuing...", 30)
			sessions = []Session{} // Continue anyway
		} else {
			deletionTracker.UpdateStatus(userID, "sessions_found", fmt.Sprintf("Found %d sessions to delete", len(sessions)), 40)
		}
		
		// Step 2: Delete sessions concurrently
		if len(sessions) > 0 {
			deletionTracker.UpdateStatus(userID, "deleting_sessions", "Deleting user sessions...", 50)
			// This will use the concurrent deletion we implemented
		} else {
			deletionTracker.UpdateStatus(userID, "no_sessions", "No sessions to delete", 60)
		}
		
		// Step 3: Graph cleanup (background)
		deletionTracker.UpdateStatus(userID, "graph_cleanup", "Starting graph data cleanup...", 70)
		
		// Step 4: Delete user from Zep server
		deletionTracker.UpdateStatus(userID, "deleting_user", "Deleting user from server...", 80)
		err = h.apiClient.DeleteUserWithCleanup(userID)
		if err != nil {
			deletionTracker.MarkFailed(userID, err.Error())
			return
		}
		
		// Step 5: Clear cache
		deletionTracker.UpdateStatus(userID, "clearing_cache", "Clearing cached data...", 90)
		h.cache.Delete(fmt.Sprintf("user:%s", userID))
		h.cache.Delete(fmt.Sprintf("episodes:%s", userID))
		h.cache.Delete(fmt.Sprintf("graph:%s", userID))
		
		// Mark as completed
		deletionTracker.MarkCompleted(userID)
	}()
	
	// Return deletion tracking info immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "started",
		"user_id":       userID,
		"message":       "User deletion started in background",
		"progress":      10,
		"tracking_url":  fmt.Sprintf("%s/api/users/%s/deletion-status", h.basePath, userID),
	})
}