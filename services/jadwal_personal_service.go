package services

import (
	"math"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/dto"
	"github.com/habbazettt/muraja-server/models"
	"github.com/habbazettt/muraja-server/utils"
	"github.com/sirupsen/logrus"
)

type JadwalPersonalService struct {
	DB *gorm.DB
}

func (s *JadwalPersonalService) GetAllJadwalPersonal(c *fiber.Ctx) error {
	log := logrus.WithField("handler", "GetAllJadwalPersonal")
	log.Info("Menerima permintaan untuk mengambil semua jadwal personal")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	kesibukan := c.Query("kesibukan", "")
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var jadwalPersonals []models.JadwalPersonal
	var totalJadwals int64

	query := s.DB.Model(&models.JadwalPersonal{})
	if kesibukan != "" {
		query = query.Where("kesibukan ILIKE ?", "%"+kesibukan+"%")
	}

	if err := query.Count(&totalJadwals).Error; err != nil {
		log.WithError(err).Error("Gagal menghitung total jadwal personal")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses permintaan", err.Error())
	}

	if err := query.Preload("User").
		Order("updated_at DESC").Limit(limit).Offset(offset).
		Find(&jadwalPersonals).Error; err != nil {
		log.WithError(err).Error("Gagal mengambil daftar jadwal personal")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal mengambil data", err.Error())
	}

	responseDTOs := make([]dto.JadwalPersonalDetailResponse, len(jadwalPersonals))
	for i, jadwal := range jadwalPersonals {
		ownerName := "N/A"
		ownerRole := "N/A"

		if jadwal.User != nil {
			ownerName = jadwal.User.Nama
			ownerRole = jadwal.User.UserType
		}

		responseDTOs[i] = dto.JadwalPersonalDetailResponse{
			ID:                jadwal.ID,
			OwnerName:         ownerName,
			OwnerRole:         ownerRole,
			TotalHafalan:      jadwal.TotalHafalan,
			Jadwal:            jadwal.Jadwal,
			Kesibukan:         jadwal.Kesibukan,
			EfektifitasJadwal: jadwal.EfektifitasJadwal,
			UpdatedAt:         jadwal.UpdatedAt,
		}
	}

	log.WithFields(logrus.Fields{
		"page":  page,
		"limit": limit,
	}).Info("Berhasil mengambil semua jadwal personal dengan pagination")

	return utils.SuccessResponse(c, fiber.StatusOK, "Semua jadwal personal berhasil diambil", fiber.Map{
		"pagination": fiber.Map{
			"current_page": page,
			"total_data":   totalJadwals,
			"total_pages":  int(math.Ceil(float64(totalJadwals) / float64(limit))),
		},
		"jadwal_personals": responseDTOs,
	})
}

func (s *JadwalPersonalService) CreateJadwalPersonal(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Invalid or missing token", nil)
	}

	userID := claims.ID
	userRole := claims.Role

	log := logrus.WithFields(logrus.Fields{"userID": userID, "userRole": userRole})

	var req dto.CreateJadwalPersonalRequest
	if err := c.BodyParser(&req); err != nil {
		log.WithError(err).Warn("Gagal mem-parsing request body untuk jadwal personal")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Request body tidak valid", err.Error())
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		jadwalPersonal := models.JadwalPersonal{
			UserID:            userID,
			TotalHafalan:      req.TotalHafalan,
			Jadwal:            req.Jadwal,
			Kesibukan:         req.Kesibukan,
			EfektifitasJadwal: req.EfektifitasJadwal,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"total_hafalan", "jadwal", "kesibukan", "efektifitas_jadwal", "updated_at"}),
		}).Create(&jadwalPersonal).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("is_data_murojaah_filled", true).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.WithError(err).Error("Gagal menyimpan jadwal personal ke database (transaksi gagal)")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal menyimpan jadwal", err.Error())
	}

	var finalJadwal models.JadwalPersonal
	s.DB.Where("user_id = ?", userID).First(&finalJadwal)

	log.Info("Jadwal personal berhasil disimpan/diperbarui dan status pengguna diperbarui")

	response := dto.JadwalPersonalResponse{
		ID:                finalJadwal.ID,
		UserID:            finalJadwal.UserID,
		TotalHafalan:      finalJadwal.TotalHafalan,
		Jadwal:            finalJadwal.Jadwal,
		Kesibukan:         finalJadwal.Kesibukan,
		EfektifitasJadwal: finalJadwal.EfektifitasJadwal,
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Jadwal personal berhasil disimpan", response)
}

func (s *JadwalPersonalService) GetJadwalPersonal(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID := claims.ID
	userRole := claims.Role

	log := logrus.WithFields(logrus.Fields{"userID": userID, "userRole": userRole})

	var jadwalPersonal models.JadwalPersonal

	err := s.DB.Where("user_id = ?", userID).First(&jadwalPersonal).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("Jadwal personal tidak ditemukan")
			return utils.SuccessResponse(c, fiber.StatusOK, "Jadwal personal tidak ditemukan", nil)
		}
		log.WithError(err).Error("Gagal mengambil jadwal personal")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal mengambil jadwal", err.Error())
	}

	response := dto.JadwalPersonalResponse{
		ID:                jadwalPersonal.ID,
		UserID:            jadwalPersonal.UserID,
		TotalHafalan:      jadwalPersonal.TotalHafalan,
		Jadwal:            jadwalPersonal.Jadwal,
		Kesibukan:         jadwalPersonal.Kesibukan,
		EfektifitasJadwal: jadwalPersonal.EfektifitasJadwal,
	}

	log.Info("Jadwal personal berhasil diambil")
	return utils.SuccessResponse(c, fiber.StatusOK, "Jadwal personal berhasil diambil", response)
}

func (s *JadwalPersonalService) UpdateJadwalPersonal(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID := claims.ID
	userRole := claims.Role

	log := logrus.WithFields(logrus.Fields{"userID": userID, "userRole": userRole})

	var req dto.UpdateJadwalPersonalRequest
	if err := c.BodyParser(&req); err != nil {
		log.WithError(err).Warn("Gagal mem-parsing request body untuk update jadwal personal")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Request body tidak valid", err.Error())
	}

	var jadwalPersonal models.JadwalPersonal
	query := s.DB.Where("user_id = ?", userID)

	if err := query.First(&jadwalPersonal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("Jadwal personal tidak ditemukan untuk diupdate")
			return utils.ResponseError(c, fiber.StatusNotFound, "Jadwal personal tidak ditemukan", nil)
		}
		log.WithError(err).Error("Gagal mencari jadwal personal")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses permintaan", err.Error())
	}

	updated := false
	if req.TotalHafalan != nil {
		jadwalPersonal.TotalHafalan = *req.TotalHafalan
		updated = true
	}
	if req.Jadwal != nil {
		jadwalPersonal.Jadwal = *req.Jadwal
		updated = true
	}
	if req.Kesibukan != nil {
		jadwalPersonal.Kesibukan = *req.Kesibukan
		updated = true
	}
	if req.EfektifitasJadwal != nil {
		jadwalPersonal.EfektifitasJadwal = *req.EfektifitasJadwal
		updated = true
	}

	if !updated {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Tidak ada data yang diubah", nil)
	}

	if err := s.DB.Save(&jadwalPersonal).Error; err != nil {
		log.WithError(err).Error("Gagal memperbarui jadwal personal di database")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memperbarui jadwal", err.Error())
	}

	log.Info("Jadwal personal berhasil diperbarui")

	response := dto.JadwalPersonalResponse{
		ID:                jadwalPersonal.ID,
		UserID:            jadwalPersonal.UserID,
		TotalHafalan:      jadwalPersonal.TotalHafalan,
		Jadwal:            jadwalPersonal.Jadwal,
		Kesibukan:         jadwalPersonal.Kesibukan,
		EfektifitasJadwal: jadwalPersonal.EfektifitasJadwal,
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Jadwal personal berhasil diperbarui", response)
}
