package route

import (
	"github.com/gofiber/fiber/v2"
	controller "github.com/sh5080/ndns-go/pkg/controllers"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
)

// SetupSearchRoutes는 검색 관련 라우트를 설정합니다
func SetupSearchRoutes(endpoint string, api fiber.Router, services *_interface.ServiceContainer) {
	api.Get(endpoint, controller.Search(services.SearchService, services.SponsorService))
}
