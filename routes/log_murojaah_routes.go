package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/middlewares"
	"github.com/habbazettt/muraja-server/services"
	"gorm.io/gorm"
)

func SetupLogMurojaahRoutes(app *fiber.App, db *gorm.DB) {
	service := services.LogMurojaahService{DB: db}

	LogRoutes := app.Group("/api/v1/log-harian", middlewares.JWTMiddleware)
	{
		LogRoutes.Get("/", service.GetOrCreateLogHarian)
		LogRoutes.Post("/detail", service.AddDetailToLog)
		LogRoutes.Put("/detail/:detailID", service.UpdateDetailLog)
		LogRoutes.Delete("/detail/:detailID", service.DeleteDetailLog)
		LogRoutes.Get("/rekap/mingguan", service.GetRecapMingguan)
		LogRoutes.Get("/statistik", service.GetStatistikMurojaah)
		LogRoutes.Post("/detail/dari-rekomendasi", service.ApplyAIRekomendasi)
	}
}
