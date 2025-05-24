package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sh5080/ndns-go/pkg/configs"
	middleware "github.com/sh5080/ndns-go/pkg/middlewares"
	route "github.com/sh5080/ndns-go/pkg/routes"
	"github.com/sh5080/ndns-go/pkg/utils"
)

func main() {
	// 메트릭 초기화
	utils.InitMetrics()

	app := fiber.New(fiber.Config{
		AppName: "NDNS-GO Service",
	})

	// 미들웨어 설정
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(middleware.Prometheus()) // 온프레미스 환경에서만 Prometheus 메트릭 수집

	// 라우트 설정
	route.SetupRoutes(app, false) // false: 서버리스 환경 아님을 표시

	// 서버 시작
	port := configs.GetConfig().Server.Port
	log.Fatal(app.Listen(":" + port))
}
