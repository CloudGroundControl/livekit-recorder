package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recording"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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

type RecordingController struct {
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

func NewRecordingController(creds LiveKitCredentials, service recording.Service) RecordingController {
	return RecordingController{creds, service}
}

var ErrEmptyFields = errors.New("one or more fields is empty")

func (rc *RecordingController) StartRecording(c echo.Context) error {
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

func (rc *RecordingController) StopRecording(c echo.Context) error {
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

func (rc *RecordingController) ReceiveWebhooks(c echo.Context) error {
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
		// Spawn a goroutine that will poll and check that the participant has at least 1 track
		go func(room string, participant string) {
			ctx := context.TODO()
			deadline := time.After(time.Second * 10)
			ticker := time.NewTicker(time.Second * 2)
			done := make(chan struct{}, 1)

			var err error
			defer func() {
				// Stop timer
				ticker.Stop()

				// Handle polling errors
				if err != nil {
					log.Errorf("webhook cannot check tracks | error: %v, participant: %s", err, event.Participant.Name)
					return
				}

				// Start recording
				var profile, err = rc.Service.SuggestMediaProfile(ctx, event.Room.Name, event.Participant.Identity)
				if err != nil {
					log.Errorf("webhook cannot suggest profile | error: %v, participant: %s", err, event.Participant.Name)
					return
				}
				err = rc.Service.StartRecording(ctx, recording.StartRecordingRequest{
					Room:        event.Room.Name,
					Participant: event.Participant.Identity,
					Profile:     profile,
				})
				if err != nil {
					log.Error("webhook cannot start recording | error: %v, participant: %s", err, event.Participant.Name)
				}
			}()

			// Block forever until participant has at least 1 track or until it's past the deadline.
			// In between ticks, it will make an API call to check for participants
			for {
				select {
				case <-done:
					return
				case <-deadline:
					err = errors.New("too long")
					return
				case <-ticker.C:
					pi, err := rc.LKRoomService().GetParticipant(ctx, &livekit.RoomParticipantIdentity{
						Room:     room,
						Identity: participant,
					})
					if err != nil {
						return
					} else if len(pi.Tracks) < 1 {
						err = errors.New("participant tracks are not available yet")
						if err != nil {
							return
						}
					}
					close(done)
				}
			}
		}(event.Room.Name, event.Participant.Identity)
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
