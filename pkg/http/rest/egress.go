package rest

import (
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
	Channel     string `json:"channel"`
	File        string `json:"file"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Sink        string `json:"sink"`
}

func NewEgressController(service egress.Service) egressController {
	return egressController{service}
}

func (ec *egressController) StartRecording(c echo.Context) error {
	// Bind request data
	data := new(StartRecordingRequest)
	if err := c.Bind(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Call service
	err := ec.Service.StartRecording(c.Request().Context(), egress.StartRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
		Channel:     egress.OutputChannel(data.Channel),
		File:        data.File,
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
