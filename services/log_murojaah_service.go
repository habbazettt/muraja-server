package services

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/dto"
	"github.com/habbazettt/muraja-server/models"
	"github.com/habbazettt/muraja-server/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type LogMurojaahService struct {
	DB *gorm.DB
}

func calculateTotalPages(startJuz, startHalaman, endJuz, endHalaman int) (int, error) {
	if startJuz > endJuz || (startJuz == endJuz && startHalaman > endHalaman) {
		return 0, errors.New("target/progres akhir tidak boleh lebih kecil dari awal")
	}

	const halamanPerJuz = 20

	if startJuz == endJuz {
		return (endHalaman - startHalaman) + 1, nil
	}

	halamanDiJuzAwal := (halamanPerJuz - startHalaman) + 1

	halamanDiJuzAkhir := endHalaman

	juzPerantara := (endJuz - startJuz) - 1
	halamanDiJuzPerantara := juzPerantara * halamanPerJuz

	return halamanDiJuzAwal + halamanDiJuzAkhir + halamanDiJuzPerantara, nil
}

func (s *LogMurojaahService) recalculateTotals(tx *gorm.DB, logHarianID uint) error {
	var totals struct {
		TotalTarget  int
		TotalSelesai int
	}

	err := tx.Model(&models.DetailLog{}).
		Select("COALESCE(SUM(total_target_halaman), 0) as total_target, COALESCE(SUM(total_selesai_halaman), 0) as total_selesai").
		Where("log_harian_id = ?", logHarianID).
		Scan(&totals).Error

	if err != nil {
		return err
	}

	return tx.Model(&models.LogHarian{}).Where("id = ?", logHarianID).Updates(map[string]interface{}{
		"total_target_halaman":  totals.TotalTarget,
		"total_selesai_halaman": totals.TotalSelesai,
	}).Error
}

func (s *LogMurojaahService) GetOrCreateLogHarian(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}

	var targetUserID uint
	if claims.Role == "admin" && c.Query("userID") != "" {
		id, err := strconv.Atoi(c.Query("userID"))
		if err != nil {
			return utils.ResponseError(c, fiber.StatusBadRequest, "Query parameter userID tidak valid", nil)
		}
		targetUserID = uint(id)
	} else {
		targetUserID = claims.ID
	}

	log := logrus.WithFields(logrus.Fields{
		"handler":      "GetOrCreateLogHarian",
		"targetUserID": targetUserID,
		"requesterID":  claims.ID,
	})

	tanggalStr := c.Query("tanggal")
	var tanggal time.Time
	var err error
	if tanggalStr == "" {
		tanggal = time.Now()
	} else {
		tanggal, err = time.Parse("2006-01-02", tanggalStr)
		if err != nil {
			log.WithError(err).Warn("Format tanggal tidak valid")
			return utils.ResponseError(c, fiber.StatusBadRequest, "Format tanggal tidak valid, gunakan YYYY-MM-DD", nil)
		}
	}
	tanggal = time.Date(tanggal.Year(), tanggal.Month(), tanggal.Day(), 0, 0, 0, 0, time.UTC)

	var logHarian models.LogHarian
	err = s.DB.Preload("DetailLogs").
		Where(models.LogHarian{UserID: targetUserID, Tanggal: tanggal}).
		FirstOrCreate(&logHarian).Error

	if err != nil {
		log.WithError(err).Error("Gagal mengambil atau membuat log harian")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses data log harian", err.Error())
	}

	detailDTOs := make([]dto.DetailLogResponse, len(logHarian.DetailLogs))
	for i, detail := range logHarian.DetailLogs {
		detailDTOs[i] = dto.DetailLogResponse{
			ID:                  detail.ID,
			WaktuMurojaah:       detail.WaktuMurojaah,
			TargetStartJuz:      detail.TargetStartJuz,
			TargetStartHalaman:  detail.TargetStartHalaman,
			TargetEndJuz:        detail.TargetEndJuz,
			TargetEndHalaman:    detail.TargetEndHalaman,
			TotalTargetHalaman:  detail.TotalTargetHalaman,
			SelesaiEndJuz:       detail.SelesaiEndJuz,
			SelesaiEndHalaman:   detail.SelesaiEndHalaman,
			TotalSelesaiHalaman: detail.TotalSelesaiHalaman,
			Status:              string(detail.Status),
			Catatan:             detail.Catatan,
			UpdatedAt:           detail.UpdatedAt,
		}
	}

	response := dto.LogHarianResponse{
		ID:                  logHarian.ID,
		UserID:              logHarian.UserID,
		Tanggal:             logHarian.Tanggal.Format("02-01-2006"),
		TotalTargetHalaman:  logHarian.TotalTargetHalaman,
		TotalSelesaiHalaman: logHarian.TotalSelesaiHalaman,
		DetailLogs:          detailDTOs,
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Log harian berhasil diproses", response)
}

func (s *LogMurojaahService) AddDetailToLog(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}
	targetUserID := claims.ID

	log := logrus.WithFields(logrus.Fields{
		"handler":     "AddDetailToLog",
		"userID":      targetUserID,
		"requesterID": claims.ID,
	})

	var req dto.AddDetailLogRequest
	if err := c.BodyParser(&req); err != nil {
		log.WithError(err).Error("Gagal parsing body request")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Request body tidak valid", err.Error())
	}

	var newDetail models.DetailLog

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		today := time.Now().UTC()
		today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

		var logHarian models.LogHarian
		if err := tx.Where(models.LogHarian{UserID: targetUserID, Tanggal: today}).FirstOrCreate(&logHarian).Error; err != nil {
			return err
		}

		totalTarget, err := calculateTotalPages(req.TargetStartJuz, req.TargetStartHalaman, req.TargetEndJuz, req.TargetEndHalaman)
		if err != nil {
			return err
		}
		if totalTarget <= 0 {
			return errors.New("target murojaah harus lebih dari 0 halaman")
		}

		newDetail = models.DetailLog{
			LogHarianID:        logHarian.ID,
			WaktuMurojaah:      req.WaktuMurojaah,
			TargetStartJuz:     req.TargetStartJuz,
			TargetStartHalaman: req.TargetStartHalaman,
			TargetEndJuz:       req.TargetEndJuz,
			TargetEndHalaman:   req.TargetEndHalaman,
			TotalTargetHalaman: totalTarget,
			Status:             models.StatusSesiBelumSelesai,
			Catatan:            req.Catatan,
		}
		if err := tx.Create(&newDetail).Error; err != nil {
			return err
		}

		return s.recalculateTotals(tx, logHarian.ID)
	})

	if err != nil {
		log.WithError(err).Error("Gagal menambahkan detail log dalam transaksi")
		if err.Error() == "target murojaah harus lebih dari 0 halaman" || err.Error() == "target/progres akhir tidak boleh lebih kecil dari awal" {
			return utils.ResponseError(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal menyimpan sesi murojaah", err.Error())
	}

	response := dto.DetailLogResponse{
		ID:                  newDetail.ID,
		WaktuMurojaah:       newDetail.WaktuMurojaah,
		TargetStartJuz:      newDetail.TargetStartJuz,
		TargetStartHalaman:  newDetail.TargetStartHalaman,
		TargetEndJuz:        newDetail.TargetEndJuz,
		TargetEndHalaman:    newDetail.TargetEndHalaman,
		TotalTargetHalaman:  newDetail.TotalTargetHalaman,
		SelesaiEndJuz:       newDetail.SelesaiEndJuz,
		SelesaiEndHalaman:   newDetail.SelesaiEndHalaman,
		TotalSelesaiHalaman: newDetail.TotalSelesaiHalaman,
		Status:              string(newDetail.Status),
		Catatan:             newDetail.Catatan,
		UpdatedAt:           newDetail.UpdatedAt,
	}

	log.Info("Berhasil menambahkan detail sesi murojaah baru")
	return utils.SuccessResponse(c, fiber.StatusCreated, "Sesi murojaah berhasil ditambahkan", response)
}

func (s *LogMurojaahService) UpdateDetailLog(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}
	userID := claims.ID

	detailID, err := c.ParamsInt("detailID")
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "ID detail log tidak valid", nil)
	}

	log := logrus.WithFields(logrus.Fields{
		"handler":     "UpdateDetailLog",
		"userID":      userID,
		"detailID":    detailID,
		"requesterID": claims.ID,
	})

	var req dto.UpdateDetailLogRequest
	if err := c.BodyParser(&req); err != nil {
		log.WithError(err).Error("Gagal parsing body request")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Request body tidak valid", err.Error())
	}

	var detailLog models.DetailLog

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Joins("JOIN log_harians ON log_harians.id = detail_logs.log_harian_id").
			Where("detail_logs.id = ? AND log_harians.user_id = ?", detailID, userID).
			First(&detailLog).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("detail log tidak ditemukan atau Anda tidak punya hak akses")
			}
			return err
		}

		totalSelesai, err := calculateTotalPages(detailLog.TargetStartJuz, detailLog.TargetStartHalaman, req.SelesaiEndJuz, req.SelesaiEndHalaman)
		if err != nil {
			return err
		}

		if totalSelesai > detailLog.TotalTargetHalaman {
			totalSelesai = detailLog.TotalTargetHalaman
		}

		var newStatus models.StatusDetailLog
		if totalSelesai >= detailLog.TotalTargetHalaman {
			newStatus = models.StatusSesiSelesai
			log.Info("Progres mencapai target. Status diatur ke 'Selesai'.")
		} else {
			newStatus = models.StatusSesiBelumSelesai
			log.Info("Progres belum mencapai target. Status tetap 'Belum Selesai'.")
		}

		detailLog.SelesaiEndJuz = req.SelesaiEndJuz
		detailLog.SelesaiEndHalaman = req.SelesaiEndHalaman
		detailLog.TotalSelesaiHalaman = totalSelesai
		detailLog.Catatan = req.Catatan
		detailLog.Status = newStatus

		if err := tx.Save(&detailLog).Error; err != nil {
			return err
		}

		return s.recalculateTotals(tx, detailLog.LogHarianID)
	})

	if err != nil {
		log.WithError(err).Error("Gagal memperbarui detail log dalam transaksi")
		if err.Error() == "target/progres akhir tidak boleh lebih kecil dari awal" {
			return utils.ResponseError(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		if err.Error() == "detail log tidak ditemukan atau Anda tidak punya hak akses" {
			return utils.ResponseError(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memperbarui sesi murojaah", err.Error())
	}

	response := dto.DetailLogResponse{
		ID:                  detailLog.ID,
		WaktuMurojaah:       detailLog.WaktuMurojaah,
		TargetStartJuz:      detailLog.TargetStartJuz,
		TargetStartHalaman:  detailLog.TargetStartHalaman,
		TargetEndJuz:        detailLog.TargetEndJuz,
		TargetEndHalaman:    detailLog.TargetEndHalaman,
		TotalTargetHalaman:  detailLog.TotalTargetHalaman,
		SelesaiEndJuz:       detailLog.SelesaiEndJuz,
		SelesaiEndHalaman:   detailLog.SelesaiEndHalaman,
		TotalSelesaiHalaman: detailLog.TotalSelesaiHalaman,
		Status:              string(detailLog.Status),
		Catatan:             detailLog.Catatan,
		UpdatedAt:           detailLog.UpdatedAt,
	}

	log.Info("Berhasil memperbarui detail sesi murojaah")
	return utils.SuccessResponse(c, fiber.StatusOK, "Sesi murojaah berhasil diperbarui", response)
}

func (s *LogMurojaahService) DeleteDetailLog(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}
	userID := claims.ID

	detailID, err := c.ParamsInt("detailID")
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "ID detail log tidak valid", nil)
	}

	log := logrus.WithFields(logrus.Fields{
		"handler":     "DeleteDetailLog",
		"userID":      userID,
		"detailID":    detailID,
		"requesterID": claims.ID,
	})

	var detailLog models.DetailLog

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Joins("JOIN log_harians ON log_harians.id = detail_logs.log_harian_id").
			Where("detail_logs.id = ? AND log_harians.user_id = ?", detailID, userID).
			First(&detailLog).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("detail log tidak ditemukan atau Anda tidak punya hak akses")
			}
			return err
		}

		logHarianID := detailLog.LogHarianID

		if err := tx.Delete(&detailLog).Error; err != nil {
			return err
		}

		return s.recalculateTotals(tx, logHarianID)
	})

	if err != nil {
		log.WithError(err).Error("Gagal menghapus detail log dalam transaksi")
		if err.Error() == "detail log tidak ditemukan atau Anda tidak punya hak akses" {
			return utils.ResponseError(c, fiber.StatusNotFound, "Detail log tidak ditemukan atau Anda tidak punya hak akses", nil)
		}
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal menghapus sesi murojaah", err.Error())
	}

	log.Info("Berhasil menghapus detail sesi murojaah")
	return utils.SuccessResponse(c, fiber.StatusOK, "Sesi murojaah berhasil dihapus", nil)
}

func (s *LogMurojaahService) GetRecapMingguan(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}

	var targetUserID uint
	if claims.Role == "admin" && c.Query("userID") != "" {
		id, err := strconv.Atoi(c.Query("userID"))
		if err != nil {
			return utils.ResponseError(c, fiber.StatusBadRequest, "Query parameter userID tidak valid", nil)
		}
		targetUserID = uint(id)
	} else {
		targetUserID = claims.ID
	}

	log := logrus.WithFields(logrus.Fields{
		"handler":      "GetRecapMingguan",
		"targetUserID": targetUserID,
		"requesterID":  claims.ID,
	})
	log.Info("Menerima permintaan untuk rekap mingguan")

	type RecapResult struct {
		Tanggal             string `json:"tanggal"`
		TotalSelesaiHalaman int    `json:"total_selesai_halaman"`
	}

	var results []RecapResult

	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -6)

	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, time.UTC)

	err := s.DB.Model(&models.LogHarian{}).
		Select("to_char(tanggal, 'DD-MM-YYYY') as tanggal, total_selesai_halaman").
		Where("user_id = ? AND tanggal BETWEEN ? AND ?", targetUserID, startDate, endDate).
		Order("tanggal ASC").
		Scan(&results).Error

	if err != nil {
		log.WithError(err).Error("Gagal mengambil data rekap mingguan")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal mengambil rekap", err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Rekap mingguan berhasil diambil", results)
}

func (s *LogMurojaahService) GetStatistikMurojaah(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}

	var targetUserID uint
	if claims.Role == "admin" && c.Query("userID") != "" {
		id, err := strconv.Atoi(c.Query("userID"))
		if err != nil {
			return utils.ResponseError(c, fiber.StatusBadRequest, "Query parameter userID tidak valid", nil)
		}
		targetUserID = uint(id)
	} else {
		targetUserID = claims.ID
	}

	log := logrus.WithFields(logrus.Fields{
		"handler":      "GetStatistikMurojaah",
		"targetUserID": targetUserID,
		"requesterID":  claims.ID,
	})
	log.Info("Menerima permintaan untuk statistik murojaah")

	var stats struct {
		TotalSelesai int
		HariAktif    int
	}

	err := s.DB.Model(&models.LogHarian{}).
		Select("SUM(total_selesai_halaman) as total_selesai, COUNT(id) as hari_aktif").
		Where("user_id = ? AND total_selesai_halaman > 0", targetUserID).
		Scan(&stats).Error
	if err != nil {
		log.WithError(err).Error("Gagal menghitung statistik dasar")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses statistik", err.Error())
	}

	var rataRata float64
	if stats.HariAktif > 0 {
		rataRata = float64(stats.TotalSelesai) / float64(stats.HariAktif)
	}

	var hariProduktif dto.RecapHarianSimple
	err = s.DB.Model(&models.LogHarian{}).
		Select("to_char(tanggal, 'DD-MM-YYYY') as tanggal, total_selesai_halaman").
		Where("user_id = ?", targetUserID).
		Order("total_selesai_halaman DESC").
		Limit(1).
		Scan(&hariProduktif).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithError(err).Error("Gagal mencari hari paling produktif")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses statistik", err.Error())
	}

	var sesiProduktif struct {
		WaktuMurojaah string
	}
	err = s.DB.Model(&models.DetailLog{}).
		Select("waktu_murojaah").
		Joins("JOIN log_harians ON log_harians.id = detail_logs.log_harian_id").
		Where("log_harians.user_id = ? AND detail_logs.status = ?", targetUserID, models.StatusSesiSelesai).
		Group("waktu_murojaah").
		Order("COUNT(detail_logs.id) DESC").
		Limit(1).
		Scan(&sesiProduktif).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithError(err).Error("Gagal mencari sesi paling produktif")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal memproses statistik", err.Error())
	}

	var hariProduktifPtr *dto.RecapHarianSimple
	if hariProduktif.Tanggal != "" {
		hariProduktifPtr = &hariProduktif
	}

	response := dto.StatistikMurojaahResponse{
		TotalSelesaiHalaman:    stats.TotalSelesai,
		TotalHariAktif:         stats.HariAktif,
		RataRataHalamanPerHari: rataRata,
		SesiPalingProduktif:    sesiProduktif.WaktuMurojaah,
		HariPalingProduktif:    hariProduktifPtr,
	}

	log.Info("Berhasil mengambil data statistik murojaah")
	return utils.SuccessResponse(c, fiber.StatusOK, "Statistik murojaah berhasil diambil", response)
}

func (s *LogMurojaahService) ApplyAIRekomendasi(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Token tidak valid atau tidak ada", nil)
	}
	userID := claims.ID

	log := logrus.WithFields(logrus.Fields{
		"handler":     "ApplyAIRekomendasi",
		"userID":      userID,
		"requesterID": claims.ID,
	})

	var req dto.ApplyAIRekomendasiRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Request body tidak valid", err.Error())
	}

	var newDetail models.DetailLog

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var rekomendasi models.JadwalRekomendasi
		if err := tx.Where("id = ? AND user_id = ?", req.RekomendasiID, userID).First(&rekomendasi).Error; err != nil {
			return errors.New("riwayat rekomendasi tidak ditemukan atau bukan milik anda")
		}

		today := time.Now().UTC()
		today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

		var logHarian models.LogHarian
		if err := tx.Where(models.LogHarian{UserID: userID, Tanggal: today}).FirstOrCreate(&logHarian).Error; err != nil {
			return err
		}

		totalTarget, err := calculateTotalPages(req.TargetStartJuz, req.TargetStartHalaman, req.TargetEndJuz, req.TargetEndHalaman)
		if err != nil {
			return err
		}

		newDetail = models.DetailLog{
			LogHarianID:        logHarian.ID,
			WaktuMurojaah:      fmt.Sprintf("AI: %s", rekomendasi.RekomendasiJadwal),
			TargetStartJuz:     req.TargetStartJuz,
			TargetStartHalaman: req.TargetStartHalaman,
			TargetEndJuz:       req.TargetEndJuz,
			TargetEndHalaman:   req.TargetEndHalaman,
			TotalTargetHalaman: totalTarget,
			Status:             models.StatusSesiBelumSelesai,
			Catatan:            req.Catatan,
		}
		if err := tx.Create(&newDetail).Error; err != nil {
			return err
		}

		return s.recalculateTotals(tx, logHarian.ID)
	})

	if err != nil {
		log.WithError(err).Error("Gagal menerapkan rekomendasi AI dalam transaksi")
		if err.Error() == "riwayat rekomendasi tidak ditemukan atau bukan milik anda" {
			return utils.ResponseError(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if err.Error() == "target/progres akhir tidak boleh lebih kecil dari awal" {
			return utils.ResponseError(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Gagal menerapkan rekomendasi", err.Error())
	}

	response := dto.DetailLogResponse{
		ID:                  newDetail.ID,
		WaktuMurojaah:       newDetail.WaktuMurojaah,
		TargetStartJuz:      newDetail.TargetStartJuz,
		TargetStartHalaman:  newDetail.TargetStartHalaman,
		TargetEndJuz:        newDetail.TargetEndJuz,
		TargetEndHalaman:    newDetail.TargetEndHalaman,
		TotalTargetHalaman:  newDetail.TotalTargetHalaman,
		SelesaiEndJuz:       newDetail.SelesaiEndJuz,
		SelesaiEndHalaman:   newDetail.SelesaiEndHalaman,
		TotalSelesaiHalaman: newDetail.TotalSelesaiHalaman,
		Status:              string(newDetail.Status),
		Catatan:             newDetail.Catatan,
		UpdatedAt:           newDetail.UpdatedAt,
	}
	return utils.SuccessResponse(c, fiber.StatusCreated, "Rekomendasi berhasil diterapkan ke log harian", response)
}