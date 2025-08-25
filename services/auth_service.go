package services

import (
	"github.com/gofiber/fiber/v2"
	"github.com/habbazettt/muraja-server/dto"
	"github.com/habbazettt/muraja-server/models"
	"github.com/habbazettt/muraja-server/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AuthService struct {
	DB *gorm.DB
}

func (s *AuthService) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		logrus.WithError(err).Error("Failed to parse request body")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	var existingUser models.User
	if err := s.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		logrus.Warn("Email already registered: ", req.Email)
		return utils.ResponseError(c, fiber.StatusConflict, "Email already registered", nil)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		logrus.WithError(err).Error("Failed to hash password")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to hash password", err.Error())
	}

	user := models.User{
		Nama:                 req.Nama,
		Email:                req.Email,
		Password:             hashedPassword,
		UserType:             req.UserType,
		IsDataMurojaahFilled: false,
	}

	if err := s.DB.Create(&user).Error; err != nil {
		logrus.WithError(err).Error("Failed to register user")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to register user", err.Error())
	}

	logrus.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"nama":      user.Nama,
		"email":     user.Email,
		"user_type": user.UserType,
		"is_filled": user.IsDataMurojaahFilled,
	}).Info("User registered successfully")

	return utils.SuccessResponse(c, fiber.StatusCreated, "user registered successfully", fiber.Map{
		"id":        user.ID,
		"nama":      user.Nama,
		"email":     user.Email,
		"user_type": user.UserType,
		"is_filled": user.IsDataMurojaahFilled,
	})
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		logrus.WithError(err).Error("Failed to parse request body")
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		logrus.Warn("Invalid email or password: ", req.Email)
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Invalid Email or password", nil)
	}

	if !utils.ComparePassword(user.Password, req.Password) {
		logrus.Warn("Invalid password for Email: ", req.Email)
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Invalid Email or password", nil)
	}

	token, err := utils.GenerateToken(user.ID, user.UserType)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate token")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to generate token", err.Error())
	}

	logrus.WithFields(logrus.Fields{
		"user_id": user.ID,
	}).Info("User logged in successfully")

	return utils.SuccessResponse(c, fiber.StatusOK, "Login successful", dto.AuthResponse{
		Token: token,
		User: dto.UserResponse{
			ID:                   user.ID,
			Nama:                 user.Nama,
			Email:                user.Email,
			UserType:             user.UserType,
			IsDataMurojaahFilled: user.IsDataMurojaahFilled,
		},
	})
}

func (s *AuthService) ForgotPassword(c *fiber.Ctx) error {
	var req dto.ForgotPasswordRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	if req.Email == "" {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Email is required", nil)
	}

	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to hash password", err.Error())
	}

	if req.Email != "" {
		var user models.User
		if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
			return utils.ResponseError(c, fiber.StatusNotFound, "User not found", nil)
		}
		user.Password = hashed
		s.DB.Save(&user)
		return utils.SuccessResponse(c, fiber.StatusOK, "Password updated successfully", map[string]interface{}{
			"email":        user.Email,
			"new_password": req.NewPassword, // Kembalikan password baru
		})
	}

	return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request", nil)
}
func (s *AuthService) GetCurrentUser(c *fiber.Ctx) error {
	userClaims, ok := c.Locals("user").(*utils.Claims)
	if !ok || userClaims == nil {
		logrus.Warn("Unauthorized access: Missing user claims")
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var response interface{}

	var user models.User
	if err := s.DB.First(&user, userClaims.ID).Error; err != nil {
		logrus.Warn("User not found: ", userClaims.ID)
		return utils.ResponseError(c, fiber.StatusNotFound, "User not found", nil)
	}

	response = dto.UserResponse{
		ID:                   user.ID,
		Nama:                 user.Nama,
		Email:                user.Email,
		UserType:             user.UserType,
		IsDataMurojaahFilled: user.IsDataMurojaahFilled,
	}

	logrus.WithFields(logrus.Fields{
		"user_id": userClaims.ID,
		"role":    userClaims.Role,
	}).Info("User data retrieved successfully")

	return utils.SuccessResponse(c, fiber.StatusOK, "User data retrieved", response)
}
