package routes

import (
	"github.com/gofiber/fiber/v2"
	controller "github.com/sh5080/ndns-go/pkg/controllers"
	service "github.com/sh5080/ndns-go/pkg/services"
)

// SetupSearchRoutes는 검색 관련 라우트를 설정합니다
func SetupSearchRoutes(endpoint string, api fiber.Router, services *service.ServiceContainer) {
	// 이미 초기화된 서비스 사용
	api.Post(endpoint, controller.Search(services.SearchService, services.SponsorDetectorService))
}
