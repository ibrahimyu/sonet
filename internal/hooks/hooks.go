package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"sonet/internal/models"

	"github.com/spf13/viper"
)

// EventType represents the type of event
type EventType string

const (
	// Event types
	EventPostCreated     EventType = "post_created"
	EventPostUpdated     EventType = "post_updated"
	EventPostDeleted     EventType = "post_deleted"
	EventCommentCreated  EventType = "comment_created"
	EventCommentUpdated  EventType = "comment_updated"
	EventCommentDeleted  EventType = "comment_deleted"
	EventReactionAdded   EventType = "reaction_added"
	EventReactionRemoved EventType = "reaction_removed"
)

// Event represents a hook event
type Event struct {
	Type      EventType   `json:"type"`
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// HookManager manages event hooks
type HookManager struct {
	handlers   map[EventType][]EventHandler
	webhookURL string
	enabled    bool
	mu         sync.RWMutex
}

// EventHandler is a function that handles an event
type EventHandler func(event Event) error

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	return &HookManager{
		handlers:   make(map[EventType][]EventHandler),
		webhookURL: viper.GetString("WEBHOOK_URL"),
		enabled:    viper.GetBool("HOOKS_ENABLED"),
	}
}

// Register registers a handler for an event type
func (h *HookManager) Register(eventType EventType, handler EventHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.handlers[eventType] == nil {
		h.handlers[eventType] = []EventHandler{handler}
	} else {
		h.handlers[eventType] = append(h.handlers[eventType], handler)
	}
}

// Trigger triggers an event
func (h *HookManager) Trigger(eventType EventType, userID string, data interface{}) {
	if !h.enabled {
		return
	}

	event := Event{
		Type:   eventType,
		UserID: userID,
		Data:   data,
	}

	// Run local handlers
	h.mu.RLock()
	handlers := h.handlers[eventType]
	h.mu.RUnlock()

	for _, handler := range handlers {
		go func(handler EventHandler) {
			_ = handler(event)
		}(handler)
	}

	// Send to webhook if configured
	if h.webhookURL != "" {
		go h.sendWebhook(event)
	}
}

// sendWebhook sends the event to the configured webhook URL
func (h *HookManager) sendWebhook(event Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	resp, err := http.Post(h.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("webhook request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// Global hook manager
var DefaultHookManager = NewHookManager()

// Convenience functions for triggering events
func TriggerPostCreated(userID string, post *models.Post) {
	DefaultHookManager.Trigger(EventPostCreated, userID, post)
}

func TriggerPostUpdated(userID string, post *models.Post) {
	DefaultHookManager.Trigger(EventPostUpdated, userID, post)
}

func TriggerPostDeleted(userID string, postID string) {
	DefaultHookManager.Trigger(EventPostDeleted, userID, map[string]string{"id": postID})
}

func TriggerCommentCreated(userID string, comment *models.Comment) {
	DefaultHookManager.Trigger(EventCommentCreated, userID, comment)
}

func TriggerCommentUpdated(userID string, comment *models.Comment) {
	DefaultHookManager.Trigger(EventCommentUpdated, userID, comment)
}

func TriggerCommentDeleted(userID string, commentID string) {
	DefaultHookManager.Trigger(EventCommentDeleted, userID, map[string]string{"id": commentID})
}

func TriggerReactionAdded(userID string, reaction *models.Reaction) {
	DefaultHookManager.Trigger(EventReactionAdded, userID, reaction)
}

func TriggerReactionRemoved(userID string, reactionID string) {
	DefaultHookManager.Trigger(EventReactionRemoved, userID, map[string]string{"id": reactionID})
}
