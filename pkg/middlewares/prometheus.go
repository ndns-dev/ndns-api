package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Prometheus 미들웨어는 HTTP 요청에 대한 메트릭을 수집합니다
func Prometheus() fiber.Handler {
	// 메트릭 초기화
	utils.InitMetrics()

	return func(c *fiber.Ctx) error {
		// 요청 시작 시간
		start := time.Now()

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 시간 계산
		duration := time.Since(start).Seconds()

		// HTTP 메트릭 기록 (요청 수, 응답 시간)
		method := string(c.Request().Header.Method())
		path := c.Route().Path
		if path == "" {
			path = "/"
		}
		status := c.Response().StatusCode()

		// 메트릭 레이블 정규화
		utils.RecordRequest(method, path, status, duration)

		return err
	}
}
