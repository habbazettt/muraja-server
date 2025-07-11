package services

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/config"
	"github.com/habbazettt/muraja-server/dto"
	"github.com/habbazettt/muraja-server/models"
	"github.com/habbazettt/muraja-server/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RekomendasiService struct {
	DB *gorm.DB
}

func (s *RekomendasiService) GetRecommendation(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)

	log := logrus.WithFields(logrus.Fields{
		"handler":  "GetRecommendation",
		"userID":   claims.ID,
		"userRole": claims.Role,
	})
	log.Info("Menerima permintaan rekomendasi jadwal")

	var req dto.RecommendationRequest
	if err := c.BodyParser(&req); err != nil {
		log.WithError(err).Error("Gagal mem-parsing request body")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Cannot parse request body", err.Error())
	}

	stateString := fmt.Sprintf("%s_%s", req.Kesibukan, req.KategoriHafalan)
	log = log.WithField("state", stateString)

	var bestAction string
	var qValue *float64
	var recType string
	var persentaseEfektif *float64

	if stateActions, ok := config.QTableModel[stateString]; ok {
		var maxQ float64 = -1.0
		isFirst := true
		for action, val := range stateActions {
			if isFirst || val > maxQ {
				maxQ = val
				bestAction = action
				isFirst = false
			}
		}
		qValue = &maxQ
		recType = "Spesifik"
	} else {
		if len(config.HistoricalBest) > 0 {
			bestAction = config.HistoricalBest[0].Jadwal
			recType = "Umum (Historis Terbaik)"
		} else {
			bestAction = "Tidak ada jadwal default"
			recType = "Tidak Ada Rekomendasi"
		}
	}

	if bestAction != "Tidak ada jadwal default" && len(config.HistoricalBest) > 0 {
		for _, info := range config.HistoricalBest {
			if info.Jadwal == bestAction {
				persen := info.PersentaseEfektif
				persentaseEfektif = &persen
				break
			}
		}
	}

	log = log.WithFields(logrus.Fields{
		"rekomendasi": bestAction,
		"tipe":        recType,
	})

	response := dto.RecommendationResponse{
		State:                     stateString,
		RekomendasiJadwal:         bestAction,
		TipeRekomendasi:           recType,
		EstimasiQValue:            qValue,
		PersentaseEfektifHistoris: persentaseEfektif,
	}

	if bestAction != "Tidak ada jadwal default" {
		rekomendasiRecord := models.JadwalRekomendasi{
			State:             stateString,
			RekomendasiJadwal: response.RekomendasiJadwal,
			TipeRekomendasi:   response.TipeRekomendasi,
			EstimasiQValue:    response.EstimasiQValue,
		}

		rekomendasiRecord.UserID = claims.ID

		if err := s.DB.Create(&rekomendasiRecord).Error; err != nil {
			log.WithError(err).Error("Gagal menyimpan riwayat rekomendasi ke database")
		} else {
			log.WithField("recordID", rekomendasiRecord.ID).Info("Riwayat rekomendasi berhasil disimpan")
			response.ID = rekomendasiRecord.ID
			response.UserID = rekomendasiRecord.UserID
		}
	}

	log.Info("Rekomendasi berhasil dikirim ke pengguna")
	return utils.SuccessResponse(c, fiber.StatusOK, "Rekomendasi berhasil dibuat", response)
}

func (s *RekomendasiService) GetAllRekomendasi(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID := claims.ID
	userRole := claims.Role

	log := logrus.WithFields(logrus.Fields{
		"handler":  "GetAllRekomendasi",
		"userID":   userID,
		"userRole": userRole,
	})
	log.Info("Menerima permintaan untuk mengambil riwayat rekomendasi")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var riwayatRekomendasi []models.JadwalRekomendasi
	var totalRiwayat int64

	query := s.DB.Model(&models.JadwalRekomendasi{}).Where("user_id = ?", userID).Order("created_at DESC")

	if err := query.Count(&totalRiwayat).Error; err != nil {
		log.WithError(err).Error("Gagal menghitung total riwayat rekomendasi")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal menghitung total data", err.Error())
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&riwayatRekomendasi).Error; err != nil {
		log.WithError(err).Error("Gagal mengambil riwayat rekomendasi dari database")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal mengambil riwayat rekomendasi", err.Error())
	}

	responseDTOs := make([]dto.RecommendationResponse, len(riwayatRekomendasi))
	for i, rec := range riwayatRekomendasi {
		var persentaseEfektif *float64
		for _, info := range config.HistoricalBest {
			if info.Jadwal == rec.RekomendasiJadwal {
				persen := info.PersentaseEfektif
				persentaseEfektif = &persen
				break
			}
		}
		responseDTOs[i] = dto.RecommendationResponse{
			ID:                        rec.ID,
			State:                     rec.State,
			UserID:                    rec.UserID,
			RekomendasiJadwal:         rec.RekomendasiJadwal,
			TipeRekomendasi:           rec.TipeRekomendasi,
			EstimasiQValue:            rec.EstimasiQValue,
			PersentaseEfektifHistoris: persentaseEfektif,
		}
	}

	log.WithFields(logrus.Fields{
		"page":       page,
		"limit":      limit,
		"total_data": totalRiwayat,
	}).Info("Berhasil mengambil riwayat rekomendasi dengan pagination")

	return utils.SuccessResponse(c, fiber.StatusOK, "Riwayat rekomendasi berhasil diambil", fiber.Map{
		"pagination": fiber.Map{
			"current_page": page,
			"total_data":   totalRiwayat,
			"total_pages":  int(math.Ceil(float64(totalRiwayat) / float64(limit))),
		},
		"riwayat_rekomendasi": responseDTOs,
	})
}

func (s *RekomendasiService) GetAllKesibukan(c *fiber.Ctx) error {
	log := logrus.WithField("handler", "GetAllKesibukan")
	log.Info("Menerima permintaan untuk mengambil semua opsi kesibukan")

	kesibukanSet := make(map[string]bool)

	for stateString := range config.QTableModel {
		lastIndex := strings.LastIndex(stateString, "_")
		if lastIndex != -1 {
			kesibukan := stateString[:lastIndex]
			kesibukanSet[kesibukan] = true
		}
	}

	uniqueKesibukan := make([]string, 0, len(kesibukanSet))
	for k := range kesibukanSet {
		uniqueKesibukan = append(uniqueKesibukan, k)
	}

	sort.Strings(uniqueKesibukan)

	log.WithField("count", len(uniqueKesibukan)).Info("Berhasil mengambil daftar kesibukan unik")

	return utils.SuccessResponse(c, fiber.StatusOK, "Daftar kesibukan berhasil diambil", uniqueKesibukan)
}
