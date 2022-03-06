package rest

import (
	"errors"
	"net/http"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recording"
	"github.com/labstack/echo/v4"
)

type recordingController struct {
	recording.Service
}

type StartRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Profile     string `json:"profile"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
}

func NewRecordingController(service recording.Service) recordingController {
	return recordingController{service}
}

var ErrEmptyFields = errors.New("one or more fields is empty")

func (rc *recordingController) StartRecording(c echo.Context) error {
	// Bind request data
	data := new(StartRecordingRequest)
	if err := c.Bind(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Sanitise request
	if data.Room == "" || data.Participant == "" || data.Profile == "" {
		return echo.NewHTTPError(http.StatusBadRequest, ErrEmptyFields)
	}

	// Parse the media profile
	profile, err := recording.ParseMediaProfile(data.Profile)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Call service
	err = rc.Service.StartRecording(c.Request().Context(), recording.StartRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
		Profile:     profile,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Return success
	return c.NoContent(http.StatusOK)
}

func (rc *recordingController) StopRecording(c echo.Context) error {
	// Bind request data
	data := new(StopRecordingRequest)
	if err := c.Bind(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Sanitise request
	if data.Room == "" || data.Participant == "" {
		return echo.NewHTTPError(http.StatusBadRequest, ErrEmptyFields)
	}

	// Call service
	err := rc.Service.StopRecording(c.Request().Context(), recording.StopRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Return success
	return c.NoContent(http.StatusOK)
}
