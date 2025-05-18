package api

import (
	"errors"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"sonet/internal/adapters"
	"sonet/internal/hooks"
	"sonet/internal/models"
)

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error string `json:"error"`
}

// ErrorHandler handles API errors
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	// Check for known error types
	if errors.Is(err, gorm.ErrRecordNotFound) {
		code = fiber.StatusNotFound
	} else if errors.Is(err, fiber.ErrBadRequest) {
		code = fiber.StatusBadRequest
	} else if errors.Is(err, fiber.ErrUnauthorized) {
		code = fiber.StatusUnauthorized
	} else if errors.Is(err, fiber.ErrForbidden) {
		code = fiber.StatusForbidden
	}

	return c.Status(code).JSON(ErrorResponse{
		Error: err.Error(),
	})
}

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, db adapters.DatabaseAdapter) {
	api := app.Group("/api")

	// Health check
	api.Get("/health", healthCheck(db))
	// Post routes
	posts := api.Group("/posts")
	posts.Post("/", createPost(db))
	posts.Get("/", listPosts(db))
	posts.Get("/search", searchPosts(db))

	// Location-based post routes
	posts.Get("/nearby", findNearbyPosts(db))
	posts.Get("/city/:cityName", listPostsByCity(db))

	// Standard post CRUD routes
	posts.Get("/:id", getPost(db))
	posts.Put("/:id", updatePost(db))
	posts.Delete("/:id", deletePost(db))

	// Comment routes
	comments := api.Group("/comments")
	comments.Post("/", createComment(db))
	comments.Get("/post/:postId", listComments(db))
	comments.Get("/:id", getComment(db))
	comments.Put("/:id", updateComment(db))
	comments.Delete("/:id", deleteComment(db))

	// Reaction routes
	reactions := api.Group("/reactions")
	reactions.Post("/", createReaction(db))
	reactions.Get("/:targetType/:targetId", listReactions(db))
	reactions.Delete("/:id", deleteReaction(db))

	// Search routes
	search := api.Group("/search")
	search.Get("/posts", searchPosts(db))
}

// getUserID extracts the user ID from the request header
func getUserID(c *fiber.Ctx) string {
	return c.Get("X-User-ID")
}

// Common pagination logic
func getPaginationParams(c *fiber.Ctx) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit", "20"))
	page, _ := strconv.Atoi(c.Query("page", "1"))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}

	offset = (page - 1) * limit
	return
}

// Health check handler
func healthCheck(db adapters.DatabaseAdapter) fiber.Handler {
	startTime := time.Now()

	return func(c *fiber.Ctx) error {
		var dbStatus string
		var dbLatency time.Duration

		// Check database connectivity
		checkStart := time.Now()
		if _, err := db.ListPosts("", 1, 0); err != nil {
			dbStatus = "error: " + err.Error()
		} else {
			dbStatus = "ok"
			dbLatency = time.Since(checkStart)
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"status":      "ok",
			"service":     "Sonet API",
			"version":     viper.GetString("VERSION"),
			"uptime":      time.Since(startTime).String(),
			"date":        time.Now().Format(time.RFC3339),
			"environment": viper.GetString("ENV"),
			"go_version":  runtime.Version(),
			"go_os":       runtime.GOOS,
			"go_arch":     runtime.GOARCH,
			"database": fiber.Map{
				"adapter":    viper.GetString("DB_ADAPTER"),
				"status":     dbStatus,
				"latency_ms": dbLatency.Milliseconds(),
			},
		})
	}
}

// PostWithAttachments is a struct for handling post creation/update with attachments
type PostWithAttachments struct {
	models.Post
	AttachmentsData []struct {
		URL  string                `json:"url"`
		Type models.AttachmentType `json:"type"`
	} `json:"attachments"`
}

// Post handlers
func createPost(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		postInput := new(PostWithAttachments)
		if err := c.BodyParser(postInput); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		post := &postInput.Post
		post.UserID = userID

		// Validate location fields if provided
		if post.Latitude != 0 || post.Longitude != 0 {
			if post.Latitude < -90 || post.Latitude > 90 {
				return fiber.NewError(fiber.StatusBadRequest, "Latitude must be between -90 and 90")
			}
			if post.Longitude < -180 || post.Longitude > 180 {
				return fiber.NewError(fiber.StatusBadRequest, "Longitude must be between -180 and 180")
			}
		}

		if err := db.CreatePost(post); err != nil {
			return err
		}

		// Create attachments if any
		for _, attachmentData := range postInput.AttachmentsData {
			postID := post.ID
			attachment := &models.Attachment{
				URL:      attachmentData.URL,
				Type:     attachmentData.Type,
				PostID:   &postID,
				Metadata: models.JSON{},
			}

			// Validate attachment
			if err := attachment.Validate(); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}

			if err := db.CreateAttachment(attachment); err != nil {
				return err
			}
			post.Attachments = append(post.Attachments, *attachment)
		}

		hooks.TriggerPostCreated(userID, post)
		return c.Status(http.StatusCreated).JSON(post)
	}
}

func getPost(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
		}

		post, err := db.GetPostByID(id)
		if err != nil {
			return err
		}

		return c.JSON(post)
	}
}

func listPosts(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit, offset := getPaginationParams(c)
		userID := c.Query("user_id", "") // Filter by user ID if provided
		page, _ := strconv.Atoi(c.Query("page", "1"))

		posts, err := db.ListPosts(userID, limit, offset)
		if err != nil {
			return err
		}

		// Return with pagination metadata
		return c.JSON(fiber.Map{
			"data": posts,
			"meta": fiber.Map{
				"page":   page,
				"limit":  limit,
				"offset": offset,
				"count":  len(posts),
			},
		})
	}
}

func updatePost(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
		}

		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		post, err := db.GetPostByID(id)
		if err != nil {
			return err
		}

		if post.UserID != userID {
			return fiber.ErrForbidden
		}

		postInput := new(PostWithAttachments)
		if err := c.BodyParser(postInput); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		updatedPost := &postInput.Post

		post.Content = updatedPost.Content
		post.Metadata = updatedPost.Metadata

		// Update location fields
		post.City = updatedPost.City
		post.Latitude = updatedPost.Latitude
		post.Longitude = updatedPost.Longitude

		// Validate location fields if provided
		if post.Latitude != 0 || post.Longitude != 0 {
			if post.Latitude < -90 || post.Latitude > 90 {
				return fiber.NewError(fiber.StatusBadRequest, "Latitude must be between -90 and 90")
			}
			if post.Longitude < -180 || post.Longitude > 180 {
				return fiber.NewError(fiber.StatusBadRequest, "Longitude must be between -180 and 180")
			}
		}

		if err := db.UpdatePost(post); err != nil {
			return err
		}

		// If new attachments are provided, replace existing ones
		if len(postInput.AttachmentsData) > 0 {
			// Delete existing attachments
			for _, attachment := range post.Attachments {
				if err := db.DeleteAttachment(attachment.ID); err != nil {
					return err
				}
			}

			// Add new attachments
			post.Attachments = nil
			for _, attachmentData := range postInput.AttachmentsData {
				postID := post.ID
				attachment := &models.Attachment{
					URL:      attachmentData.URL,
					Type:     attachmentData.Type,
					PostID:   &postID,
					Metadata: models.JSON{},
				}

				// Validate attachment
				if err := attachment.Validate(); err != nil {
					return fiber.NewError(fiber.StatusBadRequest, err.Error())
				}

				if err := db.CreateAttachment(attachment); err != nil {
					return err
				}
				post.Attachments = append(post.Attachments, *attachment)
			}
		}

		hooks.TriggerPostUpdated(userID, post)
		return c.JSON(post)
	}
}

func deletePost(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
		}

		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		post, err := db.GetPostByID(id)
		if err != nil {
			return err
		}

		if post.UserID != userID {
			return fiber.ErrForbidden
		}

		if err := db.DeletePost(id); err != nil {
			return err
		}

		hooks.TriggerPostDeleted(userID, id)
		return c.SendStatus(http.StatusNoContent)
	}
}

// CommentWithAttachment is a struct for handling comment creation/update with attachment
type CommentWithAttachment struct {
	models.Comment
	AttachmentData struct {
		URL  string                `json:"url"`
		Type models.AttachmentType `json:"type"`
	} `json:"attachment"`
	HasAttachment bool `json:"has_attachment"`
}

// Comment handlers
func createComment(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		commentInput := new(CommentWithAttachment)
		if err := c.BodyParser(commentInput); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		comment := &commentInput.Comment

		if comment.PostID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Post ID is required")
		}

		// Verify the post exists
		_, err := db.GetPostByID(comment.PostID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
		}

		// If this is a reply, verify parent comment exists
		if comment.ParentID != nil {
			_, err := db.GetCommentByID(*comment.ParentID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid parent comment ID")
			}
		}

		comment.UserID = userID
		if err := db.CreateComment(comment); err != nil {
			return err
		}

		// Create attachment if provided
		if commentInput.HasAttachment {
			commentID := comment.ID
			attachment := &models.Attachment{
				URL:       commentInput.AttachmentData.URL,
				Type:      commentInput.AttachmentData.Type,
				CommentID: &commentID,
				Metadata:  models.JSON{},
			}

			// Validate attachment
			if err := attachment.Validate(); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}

			if err := db.CreateAttachment(attachment); err != nil {
				return err
			}
			comment.Attachment = attachment
		}

		hooks.TriggerCommentCreated(userID, comment)
		return c.Status(http.StatusCreated).JSON(comment)
	}
}

func getComment(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid comment ID")
		}

		comment, err := db.GetCommentByID(id)
		if err != nil {
			return err
		}

		return c.JSON(comment)
	}
}

func listComments(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		postID := c.Params("postId")
		if postID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
		}

		limit, offset := getPaginationParams(c)
		page, _ := strconv.Atoi(c.Query("page", "1"))

		comments, err := db.ListComments(postID, limit, offset)
		if err != nil {
			return err
		}

		// Return with pagination metadata
		return c.JSON(fiber.Map{
			"data": comments,
			"meta": fiber.Map{
				"page":    page,
				"limit":   limit,
				"offset":  offset,
				"count":   len(comments),
				"post_id": postID,
			},
		})
	}
}

func updateComment(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid comment ID")
		}

		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		comment, err := db.GetCommentByID(id)
		if err != nil {
			return err
		}

		if comment.UserID != userID {
			return fiber.ErrForbidden
		}

		commentInput := new(CommentWithAttachment)
		if err := c.BodyParser(commentInput); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		updatedComment := &commentInput.Comment

		comment.Content = updatedComment.Content
		comment.Metadata = updatedComment.Metadata

		if err := db.UpdateComment(comment); err != nil {
			return err
		}

		// Handle attachment update
		if commentInput.HasAttachment {
			// Delete existing attachment if any
			if comment.Attachment != nil {
				if err := db.DeleteAttachment(comment.Attachment.ID); err != nil {
					return err
				}
			}

			// Add new attachment
			commentID := comment.ID
			attachment := &models.Attachment{
				URL:       commentInput.AttachmentData.URL,
				Type:      commentInput.AttachmentData.Type,
				CommentID: &commentID,
				Metadata:  models.JSON{},
			}

			// Validate attachment
			if err := attachment.Validate(); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}

			if err := db.CreateAttachment(attachment); err != nil {
				return err
			}
			comment.Attachment = attachment
		} else if comment.Attachment != nil {
			// Remove attachment if HasAttachment is false but there was an attachment
			if err := db.DeleteAttachment(comment.Attachment.ID); err != nil {
				return err
			}
			comment.Attachment = nil
		}

		hooks.TriggerCommentUpdated(userID, comment)
		return c.JSON(comment)
	}
}

func deleteComment(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid comment ID")
		}

		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		comment, err := db.GetCommentByID(id)
		if err != nil {
			return err
		}

		if comment.UserID != userID {
			return fiber.ErrForbidden
		}

		if err := db.DeleteComment(id); err != nil {
			return err
		}

		hooks.TriggerCommentDeleted(userID, id)
		return c.SendStatus(http.StatusNoContent)
	}
}

// Reaction handlers
func createReaction(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		reaction := new(models.Reaction)
		if err := c.BodyParser(reaction); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if reaction.TargetID == "" || reaction.TargetType == "" || reaction.Type == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Target ID, target type, and reaction type are required")
		}

		// Verify target exists
		if reaction.TargetType == "post" {
			_, err := db.GetPostByID(reaction.TargetID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
			}
		} else if reaction.TargetType == "comment" {
			_, err := db.GetCommentByID(reaction.TargetID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid comment ID")
			}
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Target type must be 'post' or 'comment'")
		}

		// Check if reaction already exists
		existingReaction, err := db.GetReaction(userID, reaction.TargetID, reaction.TargetType, reaction.Type)
		if err == nil && existingReaction != nil {
			// Reaction already exists, return it
			return c.Status(http.StatusOK).JSON(existingReaction)
		}

		reaction.UserID = userID
		if err := db.CreateReaction(reaction); err != nil {
			return err
		}

		hooks.TriggerReactionAdded(userID, reaction)
		return c.Status(http.StatusCreated).JSON(reaction)
	}
}

func listReactions(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		targetID := c.Params("targetId")
		targetType := c.Params("targetType")

		if targetID == "" || targetType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Target ID and type are required")
		}

		reactions, err := db.ListReactions(targetID, targetType)
		if err != nil {
			return err
		}

		return c.JSON(reactions)
	}
}

func deleteReaction(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid reaction ID")
		}

		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		if err := db.DeleteReaction(id); err != nil {
			return err
		}

		hooks.TriggerReactionRemoved(userID, id)
		return c.SendStatus(http.StatusNoContent)
	}
}

// Search for posts based on content
func searchPosts(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := c.Query("q")
		city := c.Query("city")
		latStr := c.Query("lat")
		lngStr := c.Query("lng")
		radiusStr := c.Query("radius")

		// At least one search criteria must be provided
		if query == "" && city == "" && (latStr == "" || lngStr == "") {
			return fiber.NewError(fiber.StatusBadRequest, "At least one search criteria (query, city, or location) is required")
		}

		limit, offset := getPaginationParams(c)
		page, _ := strconv.Atoi(c.Query("page", "1"))

		// If location parameters are provided, use geospatial search
		if latStr != "" && lngStr != "" {
			lat, err := strconv.ParseFloat(latStr, 64)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid latitude format")
			}

			lng, err := strconv.ParseFloat(lngStr, 64)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid longitude format")
			}

			radius := 10.0 // Default radius in km
			if radiusStr != "" {
				radius, err = strconv.ParseFloat(radiusStr, 64)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "Invalid radius format")
				}
			}

			posts, err := db.FindNearbyPosts(lat, lng, radius, limit, offset)
			if err != nil {
				return err
			}

			// Filter by content query if provided
			if query != "" {
				filteredPosts := []*models.Post{}
				for _, post := range posts {
					if strings.Contains(strings.ToLower(post.Content), strings.ToLower(query)) {
						filteredPosts = append(filteredPosts, post)
					}
				}
				posts = filteredPosts
			}

			return c.JSON(fiber.Map{
				"data": posts,
				"meta": fiber.Map{
					"query":  query,
					"lat":    lat,
					"lng":    lng,
					"radius": radius,
					"page":   page,
					"limit":  limit,
					"offset": offset,
					"count":  len(posts),
				},
			})
		} else if city != "" {
			// Search by city
			posts, err := db.ListPostsByCity(city, limit, offset)
			if err != nil {
				return err
			}

			// Filter by content query if provided
			if query != "" {
				filteredPosts := []*models.Post{}
				for _, post := range posts {
					if strings.Contains(strings.ToLower(post.Content), strings.ToLower(query)) {
						filteredPosts = append(filteredPosts, post)
					}
				}
				posts = filteredPosts
			}

			return c.JSON(fiber.Map{
				"data": posts,
				"meta": fiber.Map{
					"query":  query,
					"city":   city,
					"page":   page,
					"limit":  limit,
					"offset": offset,
					"count":  len(posts),
				},
			})
		} else {
			// Regular text search
			posts, err := db.SearchPosts(query, limit, offset)
			if err != nil {
				return err
			}

			return c.JSON(fiber.Map{
				"data": posts,
				"meta": fiber.Map{
					"query":  query,
					"page":   page,
					"limit":  limit,
					"offset": offset,
					"count":  len(posts),
				},
			})
		}
	}
}

// List posts by city
func listPostsByCity(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// URL-decode the city name parameter
		cityName := c.Params("cityName")
		if cityName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "City name is required")
		}

		// URL decode the city name
		var err error
		cityName, err = url.QueryUnescape(cityName)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid city name format")
		}

		limit, offset := getPaginationParams(c)
		page, _ := strconv.Atoi(c.Query("page", "1"))

		posts, err := db.ListPostsByCity(cityName, limit, offset)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{
			"data": posts,
			"meta": fiber.Map{
				"city":   cityName,
				"page":   page,
				"limit":  limit,
				"offset": offset,
				"count":  len(posts),
			},
		})
	}
}

// Find nearby posts
func findNearbyPosts(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		latStr := c.Query("lat")
		lngStr := c.Query("lng")
		radiusStr := c.Query("radius", "10") // Default 10km radius

		if latStr == "" || lngStr == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Latitude and longitude are required")
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid latitude format")
		}

		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid longitude format")
		}

		radius, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid radius format")
		}

		limit, offset := getPaginationParams(c)
		page, _ := strconv.Atoi(c.Query("page", "1"))

		posts, err := db.FindNearbyPosts(lat, lng, radius, limit, offset)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{
			"data": posts,
			"meta": fiber.Map{
				"lat":    lat,
				"lng":    lng,
				"radius": radius,
				"page":   page,
				"limit":  limit,
				"offset": offset,
				"count":  len(posts),
			},
		})
	}
}
