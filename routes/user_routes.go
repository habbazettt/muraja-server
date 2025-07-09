package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/services"
	"gorm.io/gorm"
)

func SetupUserRoutes(app *fiber.App, db *gorm.DB) {
	service := services.UserService{DB: db}

	userLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many modification requests, please try again later",
			})
		},
	})

	methodLimiter := func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodPut || c.Method() == fiber.MethodDelete {
			return userLimiter(c)
		}
		return c.Next()
	}

	mahasantriRoutes := app.Group("/api/v1/user", methodLimiter)
	{
		mahasantriRoutes.Get("/", middlewares.JWTMiddleware, middlewares.RoleMiddleware("admin"), service.GetAllUsers)
		mahasantriRoutes.Get("/:id", middlewares.JWTMiddleware, service.GetUserById)
		mahasantriRoutes.Put("/:id", middlewares.JWTMiddleware, service.UpdateUser)
		mahasantriRoutes.Delete("/:id", middlewares.JWTMiddleware, middlewares.RoleMiddleware("admin"), service.DeleteUser)
	}
}
