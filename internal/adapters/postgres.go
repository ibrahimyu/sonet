package adapters

import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sonet/internal/models"
)

// PostgresAdapter implements the DatabaseAdapter interface for PostgreSQL
type PostgresAdapter struct {
	db *gorm.DB
}

// newPostgresAdapter creates a new PostgreSQL database adapter
func newPostgresAdapter() (*PostgresAdapter, error) {
	dsn := viper.GetString("DB_CONNECTION_STRING")

	// Configure the logger
	logLevel := logger.Silent
	if viper.GetString("ENV") == "development" {
		logLevel = logger.Info
	}

	// Open the database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Enable PostGIS extension for geospatial features
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS postgis").Error; err != nil {
		return nil, fmt.Errorf("failed to enable PostGIS extension: %v", err)
	}

	// Create a spatial index for geolocation queries if it doesn't exist
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_indexes WHERE indexname = 'idx_posts_location'
			) THEN
				CREATE INDEX idx_posts_location ON posts USING GIST (
					ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
				);
			END IF;
		END $$;
	`).Error; err != nil {
		return nil, fmt.Errorf("failed to create spatial index: %v", err)
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}, &models.Reaction{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %v", err)
	}

	return &PostgresAdapter{db: db}, nil
}

// CreatePost creates a new post
func (a *PostgresAdapter) CreatePost(post *models.Post) error {
	return a.db.Create(post).Error
}

// GetPostByID retrieves a post by its ID
func (a *PostgresAdapter) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	err := a.db.First(&post, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

// ListPosts retrieves posts with pagination
func (a *PostgresAdapter) ListPosts(userID string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := a.db.Order("created_at DESC").Limit(limit).Offset(offset)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&posts).Error
	return posts, err
}

// UpdatePost updates an existing post
func (a *PostgresAdapter) UpdatePost(post *models.Post) error {
	return a.db.Save(post).Error
}

// DeletePost deletes a post and all its comments and reactions
func (a *PostgresAdapter) DeletePost(id string) error {
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
func (a *PostgresAdapter) CreateComment(comment *models.Comment) error {
	return a.db.Create(comment).Error
}

// GetCommentByID retrieves a comment by its ID
func (a *PostgresAdapter) GetCommentByID(id string) (*models.Comment, error) {
	var comment models.Comment
	err := a.db.First(&comment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListComments retrieves comments with pagination
func (a *PostgresAdapter) ListComments(postID string, limit, offset int) ([]*models.Comment, error) {
	var comments []*models.Comment
	err := a.db.Where("post_id = ?", postID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error
	return comments, err
}

// UpdateComment updates an existing comment
func (a *PostgresAdapter) UpdateComment(comment *models.Comment) error {
	return a.db.Save(comment).Error
}

// DeleteComment deletes a comment and all its reactions
func (a *PostgresAdapter) DeleteComment(id string) error {
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
func (a *PostgresAdapter) CreateReaction(reaction *models.Reaction) error {
	return a.db.Create(reaction).Error
}

// GetReaction retrieves a specific reaction
func (a *PostgresAdapter) GetReaction(userID, targetID, targetType, reactionType string) (*models.Reaction, error) {
	var reaction models.Reaction
	err := a.db.Where("user_id = ? AND target_id = ? AND target_type = ? AND type = ?",
		userID, targetID, targetType, reactionType).First(&reaction).Error
	if err != nil {
		return nil, err
	}
	return &reaction, nil
}

// ListReactions retrieves all reactions for a target
func (a *PostgresAdapter) ListReactions(targetID, targetType string) ([]*models.Reaction, error) {
	var reactions []*models.Reaction
	err := a.db.Where("target_id = ? AND target_type = ?", targetID, targetType).Find(&reactions).Error
	return reactions, err
}

// DeleteReaction deletes a reaction
func (a *PostgresAdapter) DeleteReaction(id string) error {
	return a.db.Delete(&models.Reaction{}, "id = ?", id).Error
}

// Close closes the database connection
func (a *PostgresAdapter) Close() error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SearchPosts searches for posts by content using full text search
func (a *PostgresAdapter) SearchPosts(query string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post

	// Using PostgreSQL's full-text search capabilities
	// Using plainto_tsquery for simpler, space-separated search
	err := a.db.Where("to_tsvector('english', content) @@ plainto_tsquery('english', ?)", query).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, err
}

// ListPostsByCity returns posts from a specific city
func (a *PostgresAdapter) ListPostsByCity(city string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	err := a.db.Where("city = ?", city).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	return posts, err
}

// FindNearbyPosts finds posts within a certain radius of a location
func (a *PostgresAdapter) FindNearbyPosts(lat, lng float64, radiusKm float64, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post

	// Use PostGIS ST_DWithin function with the spatial index for efficient geospatial queries
	// Convert radius from km to meters for ST_DistanceSphere
	query := `
		SELECT * FROM posts
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		AND ST_DWithin(
			ST_SetSRID(ST_MakePoint(longitude, latitude), 4326),
			ST_SetSRID(ST_MakePoint(?, ?), 4326),
			?
		)
		ORDER BY ST_DistanceSphere(
			ST_MakePoint(longitude, latitude),
			ST_MakePoint(?, ?)
		) ASC
		LIMIT ? OFFSET ?
	`

	// Convert radius from km to meters for ST_DistanceSphere
	radiusMeters := radiusKm * 1000.0

	err := a.db.Raw(query,
		lng, lat, // First point coordinates (note: longitude first in ST_MakePoint)
		radiusMeters,
		lng, lat, // Second point coordinates for the ORDER BY
		limit, offset,
	).Scan(&posts).Error

	return posts, err
}
