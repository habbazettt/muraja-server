package services

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/dto"
	"github.com/habbazettt/muraja-server/models"
	"github.com/habbazettt/muraja-server/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func (s *UserService) GetAllUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	name := c.Query("nama", "")

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var totalUser int64
	var user []models.User

	query := s.DB.Model(&models.User{})
	if name != "" {
		query = query.Where("nama ILIKE ?", "%"+name+"%")
	}
	query.Count(&totalUser)

	if err := query.
		Limit(limit).Offset(offset).
		Find(&user).Error; err != nil {
		logrus.WithError(err).Error("Failed to fetch Users")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to fetch users", err.Error())
	}

	response := make([]dto.UserResponse, len(user))
	for i, m := range user {
		response[i] = dto.UserResponse{
			ID:                   m.ID,
			Nama:                 m.Nama,
			Email:                m.Email,
			UserType:             m.UserType,
			IsDataMurojaahFilled: m.IsDataMurojaahFilled,
		}
	}

	logrus.WithFields(logrus.Fields{
		"page":  page,
		"limit": limit,
		"name":  name,
	}).Info("Paginated users retrieved successfully")

	return utils.SuccessResponse(c, fiber.StatusOK, "Users retrieved successfully", fiber.Map{
		"pagination": fiber.Map{
			"current_page": page,
			"total_data":   totalUser,
			"total_pages":  int(math.Ceil(float64(totalUser) / float64(limit))),
		},
		"users": response,
	})
}

func (s *UserService) GetUserById(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*utils.Claims)
	if !ok || claims == nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized: Invalid claims", nil)
	}

	id := c.Params("id")
	requestedID, err := strconv.Atoi(id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid ID format", nil)
	}

	if claims.Role != "admin" && claims.ID != uint(requestedID) {
		return utils.ResponseError(c, fiber.StatusForbidden, "Forbidden: You can only access your own data", nil)
	}

	var user models.User
	if err := s.DB.Preload("JadwalPersonal").First(&user, id).Error; err != nil {
		logrus.WithError(err).Warn("User not found")
		return utils.ResponseError(c, fiber.StatusNotFound, "User not found", nil)
	}

	var jadwalPersonalDTO *dto.JadwalPersonalResponse
	if user.JadwalPersonal != nil {
		jadwalPersonalDTO = &dto.JadwalPersonalResponse{
			ID:                user.JadwalPersonal.ID,
			TotalHafalan:      user.JadwalPersonal.TotalHafalan,
			Jadwal:            user.JadwalPersonal.Jadwal,
			Kesibukan:         user.JadwalPersonal.Kesibukan,
			EfektifitasJadwal: user.JadwalPersonal.EfektifitasJadwal,
		}
	}

	response := dto.UserResponse{
		ID:                   user.ID,
		Nama:                 user.Nama,
		Email:                user.Email,
		UserType:             user.UserType,
		IsDataMurojaahFilled: user.IsDataMurojaahFilled,
		JadwalPersonal:       jadwalPersonalDTO,
	}

	logrus.WithFields(logrus.Fields{
		"user_id": user.ID,
	}).Info("User retrieved successfully")

	return utils.SuccessResponse(c, fiber.StatusOK, "User found", response)
}

func (s *UserService) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := s.DB.First(&user, id).Error; err != nil {
		logrus.WithError(err).Warn("User not found")
		return utils.ResponseError(c, fiber.StatusNotFound, "User not found", nil)
	}

	// Bind request body ke DTO
	var updateRequest dto.UpdateUserRequest
	if err := c.BodyParser(&updateRequest); err != nil {
		logrus.WithError(err).Error("Failed to parse request body")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	updated := false
	if updateRequest.Nama != nil && *updateRequest.Nama != user.Nama {
		user.Nama = *updateRequest.Nama
		updated = true
	}

	if updateRequest.Email != nil && *updateRequest.Email != user.Email {
		user.Email = *updateRequest.Email
		updated = true
	}

	if updateRequest.UserType != nil && *updateRequest.UserType != user.UserType {
		user.UserType = *updateRequest.UserType
		updated = true
	}

	if !updated {
		return utils.ResponseError(c, fiber.StatusBadRequest, "No changes detected", nil)
	}

	if err := s.DB.Save(&user).Error; err != nil {
		logrus.WithError(err).Error("Failed to update user")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to update user", err.Error())
	}

	logrus.WithFields(logrus.Fields{
		"user_id": user.ID,
	}).Info("User updated successfully")

	response := dto.UserResponse{
		ID:                   user.ID,
		Nama:                 user.Nama,
		Email:                user.Email,
		UserType:             user.UserType,
		IsDataMurojaahFilled: user.IsDataMurojaahFilled,
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "User updated successfully", response)
}

func (s *UserService) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := s.DB.First(&user, id).Error; err != nil {
		logrus.WithError(err).Warn("User not found")
		return utils.ResponseError(c, fiber.StatusNotFound, "User not found", nil)
	}

	s.DB.Delete(&user)
	logrus.WithFields(logrus.Fields{
		"user_id": user.ID,
	}).Info("User deleted successfully")

	return utils.SuccessResponse(c, fiber.StatusOK, "User deleted successfully", nil)
}
