package models

import (
	"time"

	"gorm.io/gorm"
)

// AttachmentType represents the type of attachment
type AttachmentType string

const (
	// Attachment types
	AttachmentTypeImage AttachmentType = "image"
	AttachmentTypeVideo AttachmentType = "video"
	AttachmentTypeFile  AttachmentType = "file"
	AttachmentTypePost  AttachmentType = "post" // For sharing other posts
)

// Attachment represents a file or media attachment to a post or comment
type Attachment struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	URL       string         `json:"url"`
	Type      AttachmentType `json:"type"`
	PostID    *string        `json:"post_id,omitempty" gorm:"index"`
	CommentID *string        `json:"comment_id,omitempty" gorm:"index"`
	Metadata  JSON           `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time      `json:"created_at"`
}

// BeforeCreate hook for attachments to generate IDs
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = NewID()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	return nil
}

// Validate checks if the attachment has valid data
func (a *Attachment) Validate() error {
	if a.URL == "" {
		return &ValidationError{Message: "URL is required"}
	}

	if a.PostID == nil && a.CommentID == nil {
		return &ValidationError{Message: "Attachment must be associated with either a post or a comment"}
	}

	// Validate that the attachment type is valid
	switch a.Type {
	case AttachmentTypeImage, AttachmentTypeVideo, AttachmentTypeFile, AttachmentTypePost:
		return nil
	default:
		return &ValidationError{Message: "Invalid attachment type"}
	}
}

// ValidationError represents an error during model validation
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
