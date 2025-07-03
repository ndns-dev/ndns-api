package utils

import (
	"github.com/gofiber/fiber/v2"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
)

func AppError(ctx *fiber.Ctx, status int, err error, message string) error {
	return ctx.Status(status).JSON(&responseDto.ErrorResponse{
		Success: false,
		Message: message,
		Error:   err.Error(),
	})
}
