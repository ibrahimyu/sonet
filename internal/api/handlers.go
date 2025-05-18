package api

import (
	"errors"
	"net/http"
	"runtime"
	"strconv"
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

// Post handlers
func createPost(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		post := new(models.Post)
		if err := c.BodyParser(post); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		post.UserID = userID
		if err := db.CreatePost(post); err != nil {
			return err
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

		updatedPost := new(models.Post)
		if err := c.BodyParser(updatedPost); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		post.Content = updatedPost.Content
		post.ImageURL = updatedPost.ImageURL
		post.Metadata = updatedPost.Metadata

		if err := db.UpdatePost(post); err != nil {
			return err
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

// Comment handlers
func createComment(db adapters.DatabaseAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := getUserID(c)
		if userID == "" {
			return fiber.ErrUnauthorized
		}

		comment := new(models.Comment)
		if err := c.BodyParser(comment); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

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

		updatedComment := new(models.Comment)
		if err := c.BodyParser(updatedComment); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		comment.Content = updatedComment.Content
		comment.Metadata = updatedComment.Metadata

		if err := db.UpdateComment(comment); err != nil {
			return err
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
		if query == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Search query is required")
		}

		limit, offset := getPaginationParams(c)
		page, _ := strconv.Atoi(c.Query("page", "1"))

		posts, err := db.SearchPosts(query, limit, offset)
		if err != nil {
			return err
		}

		// Return with pagination metadata
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
