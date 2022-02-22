package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cloudgroundcontrol/livekit-egress/pkg/egress"
	"github.com/cloudgroundcontrol/livekit-egress/pkg/http/rest"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getEnvOrFail(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("%s not set", key)
	}
	return val
}

func main() {
	// Get env variables
	port := getEnvOrFail("APP_PORT")
	lkURL := getEnvOrFail("LIVEKIT_URL")
	lkAPIKey := getEnvOrFail("LIVEKIT_API_KEY")
	lkAPISecret := getEnvOrFail("LIVEKIT_API_SECRET")

	// Initialise egress service and controller
	service := egress.NewService(lkURL, lkAPIKey, lkAPISecret)
	controller := rest.NewEgressController(service)

	// Initialise server
	e := echo.New()

	// Attach middlewares
	e.Use(middleware.Logger())

	// Attach handlers
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to CGC")
	})
	e.GET("/health-check", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// Attach egress handlers
	e.POST("/recordings/start", controller.StartRecording)
	e.POST("/recordings/stop", controller.StopRecording)

	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}
