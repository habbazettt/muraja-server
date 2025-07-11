package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/habbazettt/muraja-server/config"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/routes"
	"github.com/sirupsen/logrus"
)

func main() {
	config.LoadEnv()
	config.InitLogger()

	db := config.ConnectDB()
	config.MigrateDB()

	if err := config.LoadQlearningModels(); err != nil {
		log.Fatalf("Gagal memuat model Q-Learning: %v", err)
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST,PUT,DELETE",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(middlewares.Logger())

	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
			})
		},
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		logrus.Info("🚀 Muraja Service API is running!")
		return c.SendString("🚀 Muraja Service API is running!")
	})

	routes.SetupAuthRoutes(app, db)
	routes.SetupUserRoutes(app, db)
	routes.SetupJadwalPersonalRoutes(app, db)
	routes.SetupRekomendasiRoutes(app, db)
	routes.SetupLogMurojaahRoutes(app, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logrus.Infof("Server berjalan di http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
