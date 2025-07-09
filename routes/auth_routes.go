package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/services"
	"gorm.io/gorm"
)

func SetupAuthRoutes(app *fiber.App, db *gorm.DB) {
	services := services.AuthService{DB: db}

	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		},
	})

	auth := app.Group("/api/v1/auth")

	auth.Use(authLimiter)

	{
		auth.Post("/register", services.Register)
		auth.Post("/login", services.Login)
		auth.Post("/forget-password", services.ForgotPassword)
		auth.Get("/me", middlewares.JWTMiddleware, services.GetCurrentUser)
	}
}
