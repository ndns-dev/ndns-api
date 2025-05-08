package controller

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
)

var Version = "dev"
var GoVersion = runtime.Version()
var startTime = time.Now()

func Health() fiber.Handler {
	return func(c *fiber.Ctx) error {
		response := responseDto.HealthResponse{
			Status:    "ok",
			Time:      time.Now(),
			Version:   Version,
			Uptime:    time.Since(startTime).String(),
			GoVersion: GoVersion,
		}
		return c.JSON(response)
	}
}
