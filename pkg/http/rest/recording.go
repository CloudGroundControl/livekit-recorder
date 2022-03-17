package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"
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

	// Call service
	err := rc.Service.StartRecording(c.Request().Context(), recording.StartRecordingRequest{
		Room:        data.Room,
		Participant: data.Participant,
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

	log.Infof("received webhook | type: %v", event.Event)

	// Handle events
	if event.GetEvent() == "participant_joined" && event.Room != nil && event.Participant != nil {
		if strings.HasPrefix(event.Participant.Identity, "RB_") {
			log.Debugf("bot has joined room | identity: %s", event.Participant.Identity)
			return c.NoContent(http.StatusOK)
		}
		log.Debugf("participant has joined room | identity: %s", event.Participant.Identity)

		// Use goroutine to poll and check that the participant is in ready state
		go func(room string, participant string) {
			var state = event.Participant.State
			var err error
			ctx := context.TODO()
			ticker := time.NewTicker(time.Second * 2)
			deadline := time.After(time.Second * 10)

			log.Debugf("participant state: %s | identity: %s", state.String(), participant)

			defer func() {
				// Stop ticker
				ticker.Stop()

				// Handle error
				if err != nil {
					log.Errorf("cannot start recording | error: %v, room: %s, participant: %s", err, room, participant)
					return
				}

				// Start recording
				log.Debugf("received start recording request | room: %s, participant: %s", room, participant)
				err = rc.Service.StartRecording(ctx, recording.StartRecordingRequest{
					Room:        room,
					Participant: participant,
				})
				if err != nil {
					log.Errorf("webhook cannot start recording | error: %v, participant: %s", err, participant)
				}
			}()

			// Check state periodically: only start recording if participant state is ACTIVE
			var p *livekit.ParticipantInfo
			for {
				select {
				case <-ticker.C:
					p, err = rc.LKRoomService().GetParticipant(ctx, &livekit.RoomParticipantIdentity{
						Room:     room,
						Identity: participant,
					})
					if err != nil {
						return
					}
					state = p.State
					log.Debugf("participant state: %s | identity: %s", state.String(), participant)

					// Stop polling only if the participant's state is ACTIVE, meaning they are already publishing video
					if state == livekit.ParticipantInfo_ACTIVE {
						return
					}
				case <-deadline:
					err = errors.New("too long")
					return
				}
			}
		}(event.Room.Name, event.Participant.Identity)
	}

	if event.GetEvent() == "participant_left" && event.Room != nil && event.Participant != nil {
		if strings.HasPrefix(event.Participant.Identity, "RB_") {
			log.Debugf("bot has left room | identity: %s", event.Participant.Identity)
			return c.NoContent(http.StatusOK)
		}
		log.Debugf("participant has left room | identity: %s", event.Participant.Identity)
	}

	if event.GetEvent() == "room_finished" && event.Room != nil {
		rc.Service.DisconnectFrom(event.Room.Name)
	}

	return c.NoContent(http.StatusOK)
}
