package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Post represents a user post
type Post struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index"`
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url,omitempty"`
	Metadata  JSON      `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Comment represents a comment on a post
type Comment struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PostID    string    `json:"post_id" gorm:"index"`
	UserID    string    `json:"user_id" gorm:"index"`
	Content   string    `json:"content"`
	ParentID  *string   `json:"parent_id,omitempty" gorm:"index"`
	Metadata  JSON      `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Reaction represents a user reaction to a post or comment
type Reaction struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"user_id" gorm:"uniqueIndex:idx_reactions_unique"`
	TargetID   string    `json:"target_id" gorm:"uniqueIndex:idx_reactions_unique"`
	TargetType string    `json:"target_type" gorm:"uniqueIndex:idx_reactions_unique"` // "post" or "comment"
	Type       string    `json:"type" gorm:"uniqueIndex:idx_reactions_unique"`        // e.g., "like", "love", "haha"
	CreatedAt  time.Time `json:"created_at"`
}

// NewID generates a new UUID string
func NewID() string {
	return uuid.New().String()
}

// BeforeCreate hook for models to generate IDs
func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = NewID()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = p.CreatedAt
	return nil
}

// BeforeCreate hook for comments to generate IDs
func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = NewID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = c.CreatedAt
	return nil
}

// BeforeCreate hook for reactions to generate IDs
func (r *Reaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = NewID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	return nil
}
