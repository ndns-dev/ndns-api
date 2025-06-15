package route

import (
	"github.com/gofiber/fiber/v2"
	controller "github.com/sh5080/ndns-go/pkg/controllers"
)

// SetupAppRoutes는 애플리케이션 관련 라우트를 설정합니다
func SetupAppRoutes(app *fiber.App) {
	// 상태 확인 API
	app.Get("/health", controller.Health())

	// 메트릭 API
	app.Get("/metrics", controller.Metrics())

	app.Get("/ngrok", controller.Ngrok())
}
