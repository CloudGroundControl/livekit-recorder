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
	Profile     MediaProfile
}

type StopRecordingRequest struct {
	Room        string
	Participant string
}

type Service interface {
	StartRecording(ctx context.Context, req StartRecordingRequest) error
	StopRecording(ctx context.Context, req StopRecordingRequest) error
	SuggestMediaProfile(ctx context.Context, room string, identity string) (MediaProfile, error)
	SetUploader(uploader upload.Uploader)
	LKRoomService() *lksdk.RoomServiceClient
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

func (s *service) StartRecording(ctx context.Context, req StartRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Obtain participant info based on identity and the room they're in
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}

	// Get tracks to subscribe to according to requested profile
	tracks := s.getProfileTracks(req.Profile, pi.Tracks)

	// Verify the desired profile with the tracks being subscribed
	err = s.verifyProfile(req.Profile, tracks)
	if err != nil {
		return err
	}

	// If profile is valid, check if there is already a bot in the room. If not, create one
	_, found := s.bots[req.Room]
	if !found {
		// Create bot
		b, err := s.createBot(req.Room, botCallback{
			RemoveSubscription: s.RemoveBotSubscriptionCallback,
			SendRecordingData:  s.SendRecordingData,
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

	// Request participant to be recorded
	b.pushParticipantRequest(req.Participant, req.Profile)

	// Extract track SIDs for updating subscription
	var sids []string
	for _, t := range tracks {
		sids = append(sids, t.Sid)
	}

	// Update subscription
	return s.updateTrackSubscriptions(ctx, UpdateTrackSubscriptionsRequest{
		Room:     req.Room,
		Identity: b.id,
		SIDs:     sids,
		Subcribe: true,
	})
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
	return b.stopRecording(req.Participant)
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

func (s *service) verifyProfile(profile MediaProfile, tracks []*livekit.TrackInfo) error {
	// Build a map with track kind
	var kind lksdk.TrackKind = ""
	tracksMap := make(map[lksdk.TrackKind]*livekit.TrackInfo)
	for _, track := range tracks {
		if track.Type == livekit.TrackType_VIDEO {
			kind = lksdk.TrackKindVideo
		} else if track.Type == livekit.TrackType_AUDIO {
			kind = lksdk.TrackKindAudio
		}
		if kind != "" {
			tracksMap[kind] = track
		}
	}

	// Check number of tracks / track type matches requested profile
	switch profile {
	case MediaVideoOnly:
		if tracksMap[lksdk.TrackKindVideo] == nil {
			return errors.New("requesting video only, but did not receive video track")
		}
	case MediaAudioOnly:
		if tracksMap[lksdk.TrackKindAudio] == nil {
			return errors.New("requesting audio only, but did not receive audio track")
		}
	case MediaMuxedAV:
		if tracksMap[lksdk.TrackKindVideo] == nil || tracksMap[lksdk.TrackKindAudio] == nil {
			return errors.New("requesting both video & audio, but did not receive complete tracks")
		}
	}

	return nil
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
		}(hook)
	}
}
