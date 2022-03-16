package main

import (
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/http/rest"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/participant"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recording"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
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
	logLevel := os.Getenv("LOG_LEVEL")
	webhookUrls := os.Getenv("WEBHOOK_URLS")

	// Get log verbosity
	var verbosity log.Lvl
	switch strings.ToLower(logLevel) {
	case "debug":
		verbosity = log.DEBUG
	case "info":
		verbosity = log.INFO
	case "warn":
		verbosity = log.WARN
	case "error":
		fallthrough
	default:
		verbosity = log.ERROR
	}
	log.SetLevel(verbosity)
	log.SetHeader("(${short_file}:${line}) ${time_rfc3339} ${level}: ")

	// Separate the webhooks by comma
	var webhooks = []string{}
	if webhookUrls != "" {
		webhooks = strings.Split(webhookUrls, ",")
	}

	// Check that ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Fatal(err)
	}

	// Check if local recordings directory exists, otherwise create one. Also need to check for the right permissions
	// Value of 0755 is obtained from https://stackoverflow.com/questions/14249467/os-mkdir-and-os-mkdirall-permissions
	// for webservers.
	stat, err := os.Stat(participant.RecordingsDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(participant.RecordingsDir, 0755)
	} else if stat.Mode() != 0755 {
		err = os.Chmod(participant.RecordingsDir, 0755)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Create S3 uploader only if the environment variables are not empty
	s3Region := os.Getenv("S3_REGION")
	s3Bucket := os.Getenv("S3_BUCKET")
	var uploader upload.Uploader
	if s3Region != "" && s3Bucket != "" {
		uploader, err = upload.NewS3Uploader(upload.S3Config{
			Region:    s3Region,
			Bucket:    s3Bucket,
			Directory: os.Getenv("S3_DIRECTORY"),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// Initialise recording service
	service, err := recording.NewService(lkURL, lkAPIKey, lkAPISecret, webhooks)
	if err != nil {
		log.Fatal(err)
	}
	service.SetUploader(uploader)

	// Initialise recording controller
	creds := rest.LiveKitCredentials{
		BaseURL:   lkURL,
		APIKey:    lkAPIKey,
		APISecret: lkAPISecret,
	}
	controller := rest.NewRecordingController(creds, service)

	// Initialise server
	e := echo.New()

	// Attach middlewares
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "(${host}) ${time_rfc3339} ${level}: ${method} ${uri} ${status} ${error}\n",
	}))

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
	e.POST("/recordings/webhooks", controller.ReceiveWebhooks)

	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}
