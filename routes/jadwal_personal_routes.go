package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/services"
	"gorm.io/gorm"
)

func SetupJadwalPersonalRoutes(app *fiber.App, db *gorm.DB) {
	service := services.JadwalPersonalService{DB: db}

	jadwalRoutes := app.Group("/api/v1/jadwal-personal", middlewares.JWTMiddleware)
	{
		jadwalRoutes.Get("/all", middlewares.RoleMiddleware("admin"), service.GetAllJadwalPersonal)
		jadwalRoutes.Get("/", service.GetJadwalPersonal)
		jadwalRoutes.Post("/", service.CreateJadwalPersonal)
		jadwalRoutes.Put("/", service.UpdateJadwalPersonal)
	}
}
