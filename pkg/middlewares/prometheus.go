package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-go/pkg/configs"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Prometheus 미들웨어는 HTTP 요청에 대한 메트릭을 수집합니다
func Prometheus() fiber.Handler {
	// 서버 이름을 설정
	serverName := configs.GetConfig().Server.AppName

	return func(c *fiber.Ctx) error {
		// 요청 시작 시간
		start := time.Now()

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 시간 계산
		duration := time.Since(start).Seconds()

		// HTTP 메트릭 기록 (요청 수, 응답 시간)
		method := c.Method()
		path := c.Route().Path
		status := c.Response().StatusCode()

		utils.RecordRequest(method, path, status, duration)

		// 서버 상태 메트릭 업데이트 (주기적으로 실행)
		// 모든 요청마다 업데이트하는 것은 비효율적이므로,
		// 일정 시간(예: 10초)마다 또는 일정 요청 수마다 업데이트
		updateServerMetrics(serverName)

		return err
	}
}

// 마지막 메트릭 업데이트 시간
var lastMetricUpdate time.Time

// updateServerMetrics는 서버 상태 메트릭을 Prometheus에 업데이트합니다
func updateServerMetrics(serverName string) {
	// 10초마다 한 번씩만 업데이트
	now := time.Now()
	if now.Sub(lastMetricUpdate) < 10*time.Second {
		return
	}

	// 현재 시간 기록
	lastMetricUpdate = now

	// 시스템 메트릭 수집
	cpuUsage, memoryUsage := utils.GetSystemMetrics()

	// 서버 부하 계산 - CPU와 메모리 사용률의 가중 평균
	load := (cpuUsage * 0.7) + (memoryUsage * 0.3)

	// 서버 건강 상태 결정
	isHealthy := true
	if cpuUsage > 0.9 || memoryUsage > 0.95 {
		isHealthy = false
	}

	// 서버 처리 용량 계산
	capacity := 1.0 - load
	if capacity < 0 {
		capacity = 0
	}

	// Prometheus 메트릭 업데이트
	utils.UpdateServerMetric(serverName, "load", load)
	healthValue := 0.0
	if isHealthy {
		healthValue = 1.0
	}
	utils.UpdateServerMetric(serverName, "healthy", healthValue)
	utils.UpdateServerMetric(serverName, "capacity", capacity)
}
