package controller

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

// Metrics는 프로메테우스 메트릭을 제공하는 핸들러입니다
func Metrics() fiber.Handler {
	// Prometheus 공식 라이브러리의 HTTP 핸들러를 사용합니다
	// 이 방식으로 메트릭 형식 관련 모든 문제를 해결할 수 있습니다
	return adaptor.HTTPHandler(promhttp.Handler())
}
