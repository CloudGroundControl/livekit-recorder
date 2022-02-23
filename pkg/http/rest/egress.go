package rest

import (
	"errors"
	"net/http"

	"github.com/cloudgroundcontrol/livekit-egress/pkg/egress"
	"github.com/labstack/echo/v4"
)

type egressController struct {
	egress.Service
}

type StartRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Output      string `json:"output"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
}

func NewEgressController(service egress.Service) egressController {
	return egressController{service}
}

var ErrEmptyFields = errors.New("one or more fields is empty")

func (ec *egressController) StartRecording(c echo.Context) error {
	// Bind request data
	data := new(StartRecordingRequest)
	if err := c.Bind(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Sanitise request
	if data.Room == "" || data.Participant == "" || data.Output == "" {
		return echo.NewHTTPError(http.StatusBadRequest, ErrEmptyFields)
	}

	// Call service
	err := ec.Service.StartRecording(c.Request().Context(), egress.StartRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
		Output:      data.Output,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Return success
	return c.NoContent(http.StatusOK)
}

func (ec *egressController) StopRecording(c echo.Context) error {
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
	err := ec.Service.StopRecording(c.Request().Context(), egress.StopRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Return success
	return c.NoContent(http.StatusOK)
}
