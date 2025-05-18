package adapters

import (
	"fmt"
	"math"

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
	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}, &models.Reaction{}, &models.Attachment{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %v", err)
	}

	return &SQLiteAdapter{db: db}, nil
}

// CreatePost creates a new post
func (a *SQLiteAdapter) CreatePost(post *models.Post) error {
	return a.db.Create(post).Error
}

// GetPostByID retrieves a post by its ID with attachments
func (a *SQLiteAdapter) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	err := a.db.First(&post, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// Load attachments
	attachments, err := a.GetAttachmentsForPost(id)
	if err != nil {
		return nil, err
	}
	post.Attachments = make([]models.Attachment, len(attachments))
	for i, attachment := range attachments {
		post.Attachments[i] = *attachment
	}

	return &post, nil
}

// ListPosts retrieves posts with pagination and attachments
func (a *SQLiteAdapter) ListPosts(userID string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := a.db.Order("created_at DESC").Limit(limit).Offset(offset)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&posts).Error
	if err != nil {
		return nil, err
	}

	// Load attachments for each post
	for _, post := range posts {
		attachments, err := a.GetAttachmentsForPost(post.ID)
		if err != nil {
			return nil, err
		}
		post.Attachments = make([]models.Attachment, len(attachments))
		for i, attachment := range attachments {
			post.Attachments[i] = *attachment
		}
	}

	return posts, nil
}

// UpdatePost updates an existing post
func (a *SQLiteAdapter) UpdatePost(post *models.Post) error {
	return a.db.Save(post).Error
}

// DeletePost deletes a post and all its comments, attachments and reactions
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

		// Delete all reactions and attachments to these comments
		for _, comment := range comments {
			if err := tx.Delete(&models.Reaction{}, "target_id = ? AND target_type = ?", comment.ID, "comment").Error; err != nil {
				return err
			}

			// Delete comment attachments
			if err := tx.Delete(&models.Attachment{}, "comment_id = ?", comment.ID).Error; err != nil {
				return err
			}
		}

		// Delete all comments
		if err := tx.Delete(&models.Comment{}, "post_id = ?", id).Error; err != nil {
			return err
		}

		// Delete all attachments for this post
		if err := tx.Delete(&models.Attachment{}, "post_id = ?", id).Error; err != nil {
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

// GetCommentByID retrieves a comment by its ID with attachment
func (a *SQLiteAdapter) GetCommentByID(id string) (*models.Comment, error) {
	var comment models.Comment
	err := a.db.First(&comment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// Load attachment if exists
	attachment, err := a.GetAttachmentForComment(id)
	if err == nil {
		comment.Attachment = attachment
	}
	// If error is record not found, that's fine - comment may not have an attachment

	return &comment, nil
}

// ListComments retrieves comments with pagination and attachments
func (a *SQLiteAdapter) ListComments(postID string, limit, offset int) ([]*models.Comment, error) {
	var comments []*models.Comment
	err := a.db.Where("post_id = ?", postID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	if err != nil {
		return nil, err
	}

	// Load attachment for each comment
	for _, comment := range comments {
		attachment, err := a.GetAttachmentForComment(comment.ID)
		if err == nil {
			comment.Attachment = attachment
		}
		// If error is record not found, that's fine - comment may not have an attachment
	}

	return comments, nil
}

// UpdateComment updates an existing comment
func (a *SQLiteAdapter) UpdateComment(comment *models.Comment) error {
	return a.db.Save(comment).Error
}

// DeleteComment deletes a comment, its attachment and all its reactions
func (a *SQLiteAdapter) DeleteComment(id string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Delete all reactions to this comment
		if err := tx.Delete(&models.Reaction{}, "target_id = ? AND target_type = ?", id, "comment").Error; err != nil {
			return err
		}

		// Delete attachment if exists
		if err := tx.Delete(&models.Attachment{}, "comment_id = ?", id).Error; err != nil {
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

// ListPostsByCity returns posts from a specific city
func (a *SQLiteAdapter) ListPostsByCity(city string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	// Use exact match rather than LIKE for city
	err := a.db.Where("city = ?", city).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	return posts, err
}

// FindNearbyPosts finds posts within a certain radius of a location
func (a *SQLiteAdapter) FindNearbyPosts(lat, lng float64, radiusKm float64, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post

	// For better performance, use a bounding box first to limit the number of records
	// A rough approximation: 1 degree of latitude ≈ 111 km
	// 1 degree of longitude ≈ 111 * cos(latitude) km
	latDelta := (radiusKm * 1.2) / 111.0 // Add 20% margin for safety
	lngDelta := (radiusKm * 1.2) / (111.0 * math.Cos(lat*math.Pi/180.0))

	// Use the bounding box to reduce the number of candidates
	err := a.db.Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Where("latitude BETWEEN ? AND ?", lat-latDelta, lat+latDelta).
		Where("longitude BETWEEN ? AND ?", lng-lngDelta, lng+lngDelta).
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}

	// Filter the candidates by accurate distance and apply pagination in memory
	const earthRadius = 6371.0 // Earth radius in kilometers
	filteredPosts := make([]*models.Post, 0)

	for _, post := range posts {
		// Calculate distance using Haversine formula
		radLat1 := lat * math.Pi / 180
		radLat2 := post.Latitude * math.Pi / 180
		deltaLng := (post.Longitude - lng) * math.Pi / 180
		deltaLat := (post.Latitude - lat) * math.Pi / 180

		a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
			math.Cos(radLat1)*math.Cos(radLat2)*
				math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
		c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		distance := earthRadius * c

		if distance <= radiusKm {
			filteredPosts = append(filteredPosts, post)
		}
	}

	// Apply pagination to the filtered results
	start := offset
	end := offset + limit
	if start >= len(filteredPosts) {
		return []*models.Post{}, nil
	}
	if end > len(filteredPosts) {
		end = len(filteredPosts)
	}

	return filteredPosts[start:end], nil
}

// Attachment methods

// CreateAttachment creates a new attachment
func (a *SQLiteAdapter) CreateAttachment(attachment *models.Attachment) error {
	return a.db.Create(attachment).Error
}

// GetAttachmentsForPost retrieves all attachments for a post
func (a *SQLiteAdapter) GetAttachmentsForPost(postID string) ([]*models.Attachment, error) {
	var attachments []*models.Attachment
	err := a.db.Where("post_id = ?", postID).Find(&attachments).Error
	return attachments, err
}

// GetAttachmentForComment retrieves an attachment for a comment
func (a *SQLiteAdapter) GetAttachmentForComment(commentID string) (*models.Attachment, error) {
	var attachment models.Attachment
	err := a.db.Where("comment_id = ?", commentID).First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// DeleteAttachment deletes an attachment
func (a *SQLiteAdapter) DeleteAttachment(id string) error {
	return a.db.Delete(&models.Attachment{}, "id = ?", id).Error
}
