package serverless

import (
	"log"
	"os"
)

func CloudRunMain() {
	app := GetApp()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
