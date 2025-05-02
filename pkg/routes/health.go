package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthResponse는 상태 확인 응답 구조체입니다
type HealthResponse struct {
	Status    string    `json:"status"`
	Time      time.Time `json:"time"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	GoVersion string    `json:"goVersion"`
}

// 서버 시작 시간
var startTime = time.Now()

// SetupHealthRoutes는 상태 확인 관련 라우트를 설정합니다
func SetupHealthRoutes(app *fiber.App) {
	// 상태 확인 API
	app.Get("/health", handleHealth)
}

// handleHealth는 서버 상태 확인 요청을 처리하는 핸들러입니다
func handleHealth(c *fiber.Ctx) error {
	response := HealthResponse{
		Status:    "ok",
		Time:      time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(startTime).String(),
		GoVersion: "go1.24", // 실제 환경에서는 runtime.Version() 사용
	}

	return c.JSON(response)
}
