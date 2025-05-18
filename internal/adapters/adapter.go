package adapters

import (
	"fmt"

	"sonet/internal/models"

	"github.com/spf13/viper"
)

// DatabaseAdapter is the interface that all database adapters must implement
type DatabaseAdapter interface {
	// Posts
	CreatePost(post *models.Post) error
	GetPostByID(id string) (*models.Post, error)
	ListPosts(userID string, limit, offset int) ([]*models.Post, error)
	SearchPosts(query string, limit, offset int) ([]*models.Post, error)
	UpdatePost(post *models.Post) error
	DeletePost(id string) error

	// Location-based queries
	ListPostsByCity(city string, limit, offset int) ([]*models.Post, error)
	FindNearbyPosts(lat, lng float64, radiusKm float64, limit, offset int) ([]*models.Post, error)

	// Comments
	CreateComment(comment *models.Comment) error
	GetCommentByID(id string) (*models.Comment, error)
	ListComments(postID string, limit, offset int) ([]*models.Comment, error)
	UpdateComment(comment *models.Comment) error
	DeleteComment(id string) error

	// Reactions
	CreateReaction(reaction *models.Reaction) error
	GetReaction(userID, targetID, targetType, reactionType string) (*models.Reaction, error)
	ListReactions(targetID, targetType string) ([]*models.Reaction, error)
	DeleteReaction(id string) error

	// Utilities
	Close() error
}

// NewDatabaseAdapter creates a new database adapter based on configuration
func NewDatabaseAdapter() (DatabaseAdapter, error) {
	adapterType := viper.GetString("DB_ADAPTER")

	switch adapterType {
	case "postgres":
		return newPostgresAdapter()
	case "sqlite":
		return newSQLiteAdapter()
	case "firestore":
		return nil, fmt.Errorf("firestore adapter not implemented yet")
	case "supabase":
		return nil, fmt.Errorf("supabase adapter not implemented yet")
	default:
		return newSQLiteAdapter() // Default to SQLite
	}
}
