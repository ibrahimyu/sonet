package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/spf13/viper"
)

// RateLimiterMiddleware creates a rate limiter middleware
func RateLimiterMiddleware() fiber.Handler {
	if !viper.GetBool("RATE_LIMIT_ENABLED") {
		// Return a no-op middleware if rate limiting is disabled
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// Get rate limit configuration
	max := viper.GetInt("RATE_LIMIT_REQUESTS")
	if max <= 0 {
		max = 100
	}

	duration := viper.GetInt("RATE_LIMIT_DURATION")
	if duration <= 0 {
		duration = 60
	}

	// Create the rate limiter
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: time.Duration(duration) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use X-User-ID as the rate limiting key if available
			// Otherwise use the remote IP
			userID := c.Get("X-User-ID")
			if userID != "" {
				return "user:" + userID
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(ErrorResponse{
				Error: "Rate limit exceeded",
			})
		},
	})
}
