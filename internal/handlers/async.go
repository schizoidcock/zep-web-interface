package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/getzep/zep-web-interface/internal/cache"
	"github.com/getzep/zep-web-interface/internal/zepapi"
)

// AsyncData represents data loading status for async endpoints
type AsyncData struct {
	Status  string      `json:"status"`  // "loading", "success", "error"
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Progress int        `json:"progress,omitempty"` // 0-100
}

// BackgroundProcessor handles async data loading
type BackgroundProcessor struct {
	cache     *cache.Cache
	apiClient *zepapi.Client
}

func NewBackgroundProcessor(apiClient *zepapi.Client) *BackgroundProcessor {
	return &BackgroundProcessor{
		cache:     cache.NewCache(),
		apiClient: apiClient,
	}
}

// UserGraphAsync handles async user graph loading
func (h *Handlers) UserGraphAsync(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	cacheKey := fmt.Sprintf("graph:%s", userID)
	
	// Check if data is already cached
	if cached, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	
	// Check if loading is in progress
	loadingKey := fmt.Sprintf("graph:loading:%s", userID)
	if _, loading := h.cache.Get(loadingKey); loading {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AsyncData{
			Status:   "loading",
			Progress: 50, // Estimated progress
		})
		return
	}
	
	// Start background loading
	h.cache.Set(loadingKey, true, 10*time.Minute)
	
	go func() {
		defer h.cache.Delete(loadingKey)
		
		log.Printf("🚀 Starting background graph load for user: %s", userID)
		
		triplets, err := h.apiClient.GetUserGraphTriplets(userID)
		if err != nil {
			log.Printf("❌ Graph load failed for user %s: %v", userID, err)
			h.cache.Set(cacheKey, AsyncData{
				Status: "error",
				Error:  err.Error(),
			}, 5*time.Minute)
			return
		}
		
		log.Printf("✅ Graph load completed for user %s: %d triplets", userID, len(triplets))
		h.cache.Set(cacheKey, AsyncData{
			Status: "success",
			Data:   triplets,
		}, 30*time.Minute) // Cache for 30 minutes
	}()
	
	// Return loading status immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AsyncData{
		Status:   "loading",
		Progress: 10,
	})
}

// UserEpisodesAsync handles async user episodes loading
func (h *Handlers) UserEpisodesAsync(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	cacheKey := fmt.Sprintf("episodes:%s", userID)
	
	// Check if data is already cached
	if cached, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	
	// Check if loading is in progress
	loadingKey := fmt.Sprintf("episodes:loading:%s", userID)
	if _, loading := h.cache.Get(loadingKey); loading {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AsyncData{
			Status:   "loading",
			Progress: 70,
		})
		return
	}
	
	// Start background loading
	h.cache.Set(loadingKey, true, 5*time.Minute)
	
	go func() {
		defer h.cache.Delete(loadingKey)
		
		log.Printf("🚀 Starting background episodes load for user: %s", userID)
		
		episodes, err := h.apiClient.GetUserEpisodes(userID)
		if err != nil {
			log.Printf("❌ Episodes load failed for user %s: %v", userID, err)
			h.cache.Set(cacheKey, AsyncData{
				Status: "error",
				Error:  err.Error(),
			}, 2*time.Minute)
			return
		}
		
		log.Printf("✅ Episodes load completed for user %s: %d episodes", userID, len(episodes))
		h.cache.Set(cacheKey, AsyncData{
			Status: "success",
			Data:   episodes,
		}, 15*time.Minute) // Cache for 15 minutes
	}()
	
	// Return loading status immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AsyncData{
		Status:   "loading",
		Progress: 20,
	})
}

// Add cache to main handlers struct
func (h *Handlers) SetCache(c *cache.Cache) {
	h.cache = c
}