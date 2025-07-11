package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/services"
	"gorm.io/gorm"
)

func SetupRekomendasiRoutes(app *fiber.App, db *gorm.DB) {
	service := services.RekomendasiService{DB: db}

	rekomendasiLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many write requests, please try again later",
			})
		},
		SkipSuccessfulRequests: true,
	})

	methodLimiter := func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodPost {
			return rekomendasiLimiter(c)
		}
		return c.Next()
	}

	rekomendasiRoutes := app.Group("/api/v1/rekomendasi", middlewares.JWTMiddleware, methodLimiter)
	{
		rekomendasiRoutes.Post("/", service.GetRecommendation)
		rekomendasiRoutes.Get("/", service.GetAllRekomendasi)
		rekomendasiRoutes.Get("/kesibukan", middlewares.RoleMiddleware("admin"), service.GetAllKesibukan)
	}
}
