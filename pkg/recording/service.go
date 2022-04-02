package recording

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/participant"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/utils"
	lksdk "github.com/livekit/server-sdk-go"
)

type StartRecordingRequest struct {
	Room        string
	Participant string
}

type StopRecordingRequest struct {
	Room        string
	Participant string
}

type Service interface {
	StartRecording(ctx context.Context, req StartRecordingRequest) error
	StopRecording(ctx context.Context, req StopRecordingRequest) error
	SetUploader(uploader upload.Uploader)
	LKRoomService() *lksdk.RoomServiceClient
	DisconnectFrom(room string)
}

type service struct {
	// Info
	url string

	// State
	lock sync.Mutex
	bots map[string]*bot

	// Services
	auth     *authProvider
	lksvc    *lksdk.RoomServiceClient
	uploader upload.Uploader
	webhooks []string
}

func httpUrlFromWS(url string) string {
	if strings.Contains(url, "ws://") {
		return strings.ReplaceAll(url, "ws://", "http://")
	} else if strings.Contains(url, "wss://") {
		return strings.ReplaceAll(url, "wss://", "https://")
	}
	return ""
}

func NewService(url string, apiKey string, apiSecret string, webhooks []string) (Service, error) {
	auth := createAuthProvider(apiKey, apiSecret)
	httpUrl := httpUrlFromWS(url)
	if httpUrl == "" {
		return nil, errors.New("url must contain ws:// or wss://")
	}
	lksvc := lksdk.NewRoomServiceClient(httpUrl, apiKey, apiSecret)
	return &service{
		url:      url,
		lock:     sync.Mutex{},
		bots:     make(map[string]*bot),
		auth:     auth,
		lksvc:    lksvc,
		webhooks: webhooks,
	}, nil
}

func (s *service) SetUploader(uploader upload.Uploader) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.uploader = uploader
}

func (s *service) LKRoomService() *lksdk.RoomServiceClient {
	return s.lksvc
}

func (s *service) DisconnectFrom(room string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.bots[room]
	if !ok {
		return
	}

	b := s.bots[room]
	b.disconnect()
	delete(s.bots, room)
}

func (s *service) StartRecording(ctx context.Context, req StartRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If profile is valid, check if there is already a bot in the room. If not, create one
	_, found := s.bots[req.Room]
	if !found {
		// Create bot
		log.Debugf("no bot found in room, creating one | room: %s", req.Room)
		b, err := s.createBot(req.Room, botCallback{
			SendRecordingData: s.SendRecordingData,
		})
		if err != nil {
			return err
		}

		// Set dependencies
		b.SetUploader(s.uploader)

		// Attach the bot
		s.bots[req.Room] = b
	}

	// Retrieve the bot
	b := s.bots[req.Room]

	// Ensure that the bot can see all the tracks
	go func() {
		ctx := context.TODO()
		deadline := time.After(time.Minute * 5)
		ticker := time.NewTicker(time.Second * 2)

		var err error
		defer func() {
			// Stop ticker
			ticker.Stop()

			// Handle errors
			if err != nil {
				log.Errorf("cannot start recording | error: %v, participant: %s", err, req.Participant)
			}
		}()

		var pi *livekit.ParticipantInfo
		for {
			select {
			case <-deadline:
				err = errors.New("too long")
				return
			case <-ticker.C:
				// Get participant info
				pi, err = s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
					Room:     req.Room,
					Identity: req.Participant,
				})
				if err != nil {
					return
				}
				log.Debugf("participant exists | participant: %s, num tracks: %d", pi.Identity, len(pi.Tracks))

				// Determine media profile
				tracksMap := make(map[livekit.TrackType]bool)
				tracksSid := []string{}
				for _, t := range pi.Tracks {
					tracksMap[t.Type] = true
					tracksSid = append(tracksSid, t.Sid)
				}
				var profile MediaProfile
				if tracksMap[livekit.TrackType_VIDEO] && tracksMap[livekit.TrackType_AUDIO] {
					profile = MediaMuxedAV
				} else if tracksMap[livekit.TrackType_VIDEO] {
					profile = MediaVideoOnly
				} else if tracksMap[livekit.TrackType_AUDIO] {
					profile = MediaAudioOnly
				} else {
					log.Debugf("not seeing any tracks yet | participant: %s", req.Participant)
					continue
				}

				// Request participant to be recorded
				b.pushParticipantRequest(req.Participant, profile)

				// Update subscription
				err = s.updateTrackSubscriptions(ctx, UpdateTrackSubscriptionsRequest{
					Room:     req.Room,
					Identity: b.id,
					SIDs:     tracksSid,
					Subcribe: true,
				})
				return
			}
		}
	}()

	return nil
}

func (s *service) StopRecording(ctx context.Context, req StopRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Check bot exists
	_, found := s.bots[req.Room]
	if !found {
		return errors.New("room is not recorded")
	}

	// Retrieve bot
	b := s.bots[req.Room]

	// Stop recorder
	b.stopRecording(req.Participant)

	// Get track SIDs for the participant
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}
	var trackSids []string
	for _, t := range pi.Tracks {
		trackSids = append(trackSids, t.Sid)
	}

	// Remove subscription
	return s.updateTrackSubscriptions(ctx, UpdateTrackSubscriptionsRequest{
		Room:     req.Room,
		Identity: b.id,
		SIDs:     trackSids,
		Subcribe: false,
	})
}

func (s *service) createBot(room string, callback botCallback) (*bot, error) {
	id := utils.NewGuid("RB_")

	// Create token to join
	token, err := s.auth.buildEmptyToken(room, id)
	if err != nil {
		return nil, err
	}

	// Create bot
	return createBot(id, s.url, token, callback)
}

func (s *service) SendRecordingData(data participant.ParticipantData) {
	// Marshal to JSON
	var err error
	var body []byte
	body, err = json.Marshal(data)
	if err != nil {
		log.Errorf("error marshalling payload | error: %v, data %v", err, data)
		return
	}
	buffer := bytes.NewBuffer(body)

	// Send data
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	for _, hook := range s.webhooks {
		go func(url string) {
			_, err = client.Post(url, "application/json", buffer)
			if err != nil {
				log.Errorf("error reaching webhook | error: %v, url: %s", err, url)
			}
			log.Infof("sent webhook data | url: %s, data: %v", url, data)
		}(hook)
	}
}
