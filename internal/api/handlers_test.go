package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sonet/internal/api"
	"sonet/internal/models"
)

// MockDatabaseAdapter implements adapters.DatabaseAdapter for testing
type MockDatabaseAdapter struct {
	mock.Mock
}

func (m *MockDatabaseAdapter) CreatePost(post *models.Post) error {
	args := m.Called(post)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) GetPostByID(id string) (*models.Post, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockDatabaseAdapter) ListPosts(userID string, limit, offset int) ([]*models.Post, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockDatabaseAdapter) UpdatePost(post *models.Post) error {
	args := m.Called(post)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) DeletePost(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) CreateComment(comment *models.Comment) error {
	args := m.Called(comment)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) GetCommentByID(id string) (*models.Comment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockDatabaseAdapter) ListComments(postID string, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(postID, limit, offset)
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockDatabaseAdapter) UpdateComment(comment *models.Comment) error {
	args := m.Called(comment)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) DeleteComment(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) CreateReaction(reaction *models.Reaction) error {
	args := m.Called(reaction)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) GetReaction(userID, targetID, targetType, reactionType string) (*models.Reaction, error) {
	args := m.Called(userID, targetID, targetType, reactionType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reaction), args.Error(1)
}

func (m *MockDatabaseAdapter) ListReactions(targetID, targetType string) ([]*models.Reaction, error) {
	args := m.Called(targetID, targetType)
	return args.Get(0).([]*models.Reaction), args.Error(1)
}

func (m *MockDatabaseAdapter) DeleteReaction(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDatabaseAdapter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabaseAdapter) SearchPosts(query string, limit, offset int) ([]*models.Post, error) {
	args := m.Called(query, limit, offset)
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockDatabaseAdapter) ListPostsByCity(city string, limit, offset int) ([]*models.Post, error) {
	args := m.Called(city, limit, offset)
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockDatabaseAdapter) FindNearbyPosts(lat, lng float64, radiusKm float64, limit, offset int) ([]*models.Post, error) {
	args := m.Called(lat, lng, radiusKm, limit, offset)
	return args.Get(0).([]*models.Post), args.Error(1)
}

// Helper function to create a test app
func setupTestApp(db *MockDatabaseAdapter) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: api.ErrorHandler,
	})
	api.SetupRoutes(app, db)
	return app
}

// Test creating a post
func TestCreatePost(t *testing.T) {
	// Setup
	mockDB := new(MockDatabaseAdapter)
	app := setupTestApp(mockDB)

	// Test data
	userID := "test-user-123"
	postID := uuid.New().String()

	// Mock behavior
	mockDB.On("CreatePost", mock.MatchedBy(func(post *models.Post) bool {
		// Set ID for the returned post
		post.ID = postID
		return post.UserID == userID && post.Content == "Test post"
	})).Return(nil)

	// Make request
	body := `{"content":"Test post","metadata":{"test":"value"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)

	// Execute
	resp, err := app.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse response
	var result map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &result)

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, postID, result["id"])
	assert.Equal(t, userID, result["user_id"])
	assert.Equal(t, "Test post", result["content"])

	// Verify mocks
	mockDB.AssertExpectations(t)
}
