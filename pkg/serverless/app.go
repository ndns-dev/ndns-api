package serverless

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	route "github.com/sh5080/ndns-go/pkg/routes"
)

var app *fiber.App

// 서버리스 환경에서는 전역 변수로 앱 인스턴스를 유지하여 콜드 스타트를 최소화합니다
func init() {
	app = fiber.New(fiber.Config{
		AppName:               "NDNS-GO Service",
		DisableStartupMessage: true, // 서버리스 환경에서는 시작 메시지 비활성화
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	route.SetupRoutes(app, true) // true: 서버리스 환경임을 표시
}

// GetApp 함수는 초기화된 애플리케이션 인스턴스를 반환합니다
// 이 함수는 AWS Lambda 핸들러 또는 GCP Cloud Run 핸들러에서 호출될 수 있습니다
func GetApp() *fiber.App {
	return app
}
