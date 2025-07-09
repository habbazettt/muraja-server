package utils

import "github.com/gofiber/fiber/v2"

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func SuccessResponse(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"status":  true,
		"message": message,
		"data":    data,
	})
}

func ResponseError(c *fiber.Ctx, statusCode int, message string, details interface{}) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"status":  false,
		"message": message,
		"error":   details,
	})
}

type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalData   int `json:"total_data"`
	TotalPages  int `json:"total_pages"`
}

type ErrorResponse struct {
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}
