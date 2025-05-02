package routes

import (
	"github.com/gofiber/fiber/v2"
	service "github.com/sh5080/ndns-go/pkg/services"
)

// SetupRoutes는 애플리케이션의 모든 라우트를 설정합니다
func SetupRoutes(app *fiber.App) {
	// API 라우트 그룹
	api := app.Group("/api/v1")
	services := service.NewServiceContainer()

	// 도메인별 라우트 설정
	SetupSearchRoutes("/search", api, services)
	SetupHealthRoutes(app)
}
