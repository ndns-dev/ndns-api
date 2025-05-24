package route

import (
	"github.com/gofiber/fiber/v2"
	service "github.com/sh5080/ndns-go/pkg/services"
)

// SetupRoutes는 애플리케이션의 모든 라우트를 설정합니다
func SetupRoutes(app *fiber.App, isServerless bool) {
	// API 라우트 그룹
	api := app.Group("/api/v1")
	services := service.NewServiceContainer()

	// 도메인별 라우트 설정
	SetupSearchRoutes("/search", api, services)

	// 환경별 라우트 설정
	if !isServerless {
		// 온프레미스/VM 환경에서만 앱 라우트 설정 (헬스체크, 메트릭스 등)
		SetupAppRoutes(app)

	} else {
		// 서버리스 환경 작동 조건
	}
}
