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
	utils.InitMetrics()

	app := fiber.New(fiber.Config{
		AppName: "NDNS-GO Service",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(middleware.Prometheus())

	route.SetupRoutes(app)

	port := configs.GetConfig().Server.Port

	log.Fatal(app.Listen(":" + port))
}
