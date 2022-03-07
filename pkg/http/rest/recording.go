package rest

import (
	"errors"
	"net/http"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recording"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"google.golang.org/protobuf/encoding/protojson"
)

type LiveKitCredentials struct {
	BaseURL   string
	APIKey    string
	APISecret string
}

type recordingController struct {
	creds LiveKitCredentials
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

func NewRecordingController(creds LiveKitCredentials, service recording.Service) recordingController {
	return recordingController{creds, service}
}

var ErrEmptyFields = errors.New("one or more fields is empty")

func (rc *recordingController) StartRecording(c echo.Context) error {
	// Bind request data
	data := new(StartRecordingRequest)
	if err := c.Bind(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Sanitise request
	if data.Room == "" || data.Participant == "" {
		return echo.NewHTTPError(http.StatusBadRequest, ErrEmptyFields)
	}

	// Get profile
	var profile recording.MediaProfile
	var err error
	if data.Profile != "" {
		profile, err = recording.ParseMediaProfile(data.Profile)

	} else {
		profile, err = rc.Service.SuggestMediaProfile(c.Request().Context(), data.Room, data.Participant)
	}
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

func (rc *recordingController) ReceiveWebhooks(c echo.Context) error {
	authProvider := auth.NewFileBasedKeyProviderFromMap(map[string]string{
		rc.creds.APIKey: rc.creds.APISecret,
	})

	data, err := webhook.Receive(c.Request(), authProvider)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	event := livekit.WebhookEvent{}
	if err = protojson.Unmarshal(data, &event); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Handle events
	if event.GetEvent() == "participant_joined" && event.Room != nil && event.Participant != nil {
		profile, err := rc.Service.SuggestMediaProfile(c.Request().Context(), event.Room.Name, event.Participant.Identity)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		err = rc.Service.StartRecording(c.Request().Context(), recording.StartRecordingRequest{
			Room:        event.Room.Name,
			Participant: event.Participant.Identity,
			Profile:     profile,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
	}

	if event.GetEvent() == "participant_left" && event.Room != nil && event.Participant != nil {
		err := rc.Service.StopRecording(c.Request().Context(), recording.StopRecordingRequest{
			Room:        event.Room.Name,
			Participant: event.Participant.Identity,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
	}

	return c.NoContent(http.StatusOK)
}
