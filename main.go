package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/egress"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/http/rest"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
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
	debugMode := os.Getenv("APP_DEBUG")

	// Check that ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Fatal(err)
	}

	// Create S3 uploader only if the environment variables are not empty
	s3Region := os.Getenv("S3_REGION")
	s3Bucket := os.Getenv("S3_BUCKET")
	var uploader upload.Uploader
	if s3Region != "" && s3Bucket != "" {
		uploader, err = upload.NewS3Uploader(upload.S3Config{
			Region: s3Region,
			Bucket: s3Bucket,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// Initialise egress service and controller
	service, err := egress.NewService(lkURL, lkAPIKey, lkAPISecret, uploader)
	if err != nil {
		log.Fatal(err)
	}
	controller := rest.NewEgressController(service)

	// Initialise server
	e := echo.New()

	// Attach middlewares
	if debugMode == "true" {
		e.Use(middleware.Logger())
	}

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
