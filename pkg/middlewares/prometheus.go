package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Prometheus는 HTTP 메트릭을 수집하는 미들웨어입니다
func Prometheus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// 다음 핸들러로 이동
		err := c.Next()

		// 요청 처리 완료 후 메트릭 기록
		duration := time.Since(start).Seconds()
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Route().Path

		// 디버깅 로그
		fmt.Printf("메트릭 기록: %s %s %d (%.3f초)\n", method, path, status, duration)

		// 요청 카운터 증가
		utils.RequestCounter.WithLabelValues(method, path, strconv.Itoa(status)).Inc()

		// 응답 시간 기록
		utils.ResponseTime.WithLabelValues(method, path, strconv.Itoa(status)).Observe(duration)

		return err
	}
}
