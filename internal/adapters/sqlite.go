package adapters

import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sonet/internal/models"
)

// SQLiteAdapter implements the DatabaseAdapter interface for SQLite
type SQLiteAdapter struct {
	db *gorm.DB
}

// newSQLiteAdapter creates a new SQLite database adapter
func newSQLiteAdapter() (*SQLiteAdapter, error) {
	dbPath := viper.GetString("DB_CONNECTION_STRING")

	// Configure the logger
	logLevel := logger.Silent
	if viper.GetString("ENV") == "development" {
		logLevel = logger.Info
	}

	// Open the database connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}, &models.Reaction{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %v", err)
	}

	return &SQLiteAdapter{db: db}, nil
}

// CreatePost creates a new post
func (a *SQLiteAdapter) CreatePost(post *models.Post) error {
	return a.db.Create(post).Error
}

// GetPostByID retrieves a post by its ID
func (a *SQLiteAdapter) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	err := a.db.First(&post, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

// ListPosts retrieves posts with pagination
func (a *SQLiteAdapter) ListPosts(userID string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := a.db.Order("created_at DESC").Limit(limit).Offset(offset)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&posts).Error
	return posts, err
}

// UpdatePost updates an existing post
func (a *SQLiteAdapter) UpdatePost(post *models.Post) error {
	return a.db.Save(post).Error
}

// DeletePost deletes a post and all its comments and reactions
func (a *SQLiteAdapter) DeletePost(id string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Delete all reactions to this post
		if err := tx.Delete(&models.Reaction{}, "target_id = ? AND target_type = ?", id, "post").Error; err != nil {
			return err
		}

		// Get all comments for this post
		var comments []*models.Comment
		if err := tx.Where("post_id = ?", id).Find(&comments).Error; err != nil {
			return err
		}

		// Delete all reactions to these comments
		for _, comment := range comments {
			if err := tx.Delete(&models.Reaction{}, "target_id = ? AND target_type = ?", comment.ID, "comment").Error; err != nil {
				return err
			}
		}

		// Delete all comments
		if err := tx.Delete(&models.Comment{}, "post_id = ?", id).Error; err != nil {
			return err
		}

		// Finally delete the post
		return tx.Delete(&models.Post{}, "id = ?", id).Error
	})
}

// CreateComment creates a new comment
func (a *SQLiteAdapter) CreateComment(comment *models.Comment) error {
	return a.db.Create(comment).Error
}

// GetCommentByID retrieves a comment by its ID
func (a *SQLiteAdapter) GetCommentByID(id string) (*models.Comment, error) {
	var comment models.Comment
	err := a.db.First(&comment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListComments retrieves comments with pagination
func (a *SQLiteAdapter) ListComments(postID string, limit, offset int) ([]*models.Comment, error) {
	var comments []*models.Comment
	err := a.db.Where("post_id = ?", postID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error
	return comments, err
}

// UpdateComment updates an existing comment
func (a *SQLiteAdapter) UpdateComment(comment *models.Comment) error {
	return a.db.Save(comment).Error
}

// DeleteComment deletes a comment and all its reactions
func (a *SQLiteAdapter) DeleteComment(id string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Delete all reactions to this comment
		if err := tx.Delete(&models.Reaction{}, "target_id = ? AND target_type = ?", id, "comment").Error; err != nil {
			return err
		}

		// Delete the comment
		return tx.Delete(&models.Comment{}, "id = ?", id).Error
	})
}

// CreateReaction creates a new reaction
func (a *SQLiteAdapter) CreateReaction(reaction *models.Reaction) error {
	return a.db.Create(reaction).Error
}

// GetReaction retrieves a specific reaction
func (a *SQLiteAdapter) GetReaction(userID, targetID, targetType, reactionType string) (*models.Reaction, error) {
	var reaction models.Reaction
	err := a.db.Where("user_id = ? AND target_id = ? AND target_type = ? AND type = ?",
		userID, targetID, targetType, reactionType).First(&reaction).Error
	if err != nil {
		return nil, err
	}
	return &reaction, nil
}

// ListReactions retrieves all reactions for a target
func (a *SQLiteAdapter) ListReactions(targetID, targetType string) ([]*models.Reaction, error) {
	var reactions []*models.Reaction
	err := a.db.Where("target_id = ? AND target_type = ?", targetID, targetType).Find(&reactions).Error
	return reactions, err
}

// DeleteReaction deletes a reaction
func (a *SQLiteAdapter) DeleteReaction(id string) error {
	return a.db.Delete(&models.Reaction{}, "id = ?", id).Error
}

// Close closes the database connection
func (a *SQLiteAdapter) Close() error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SearchPosts searches for posts by content
func (a *SQLiteAdapter) SearchPosts(query string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post

	// Using LIKE for basic search functionality
	searchQuery := "%" + query + "%"

	err := a.db.Where("content LIKE ?", searchQuery).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, err
}
