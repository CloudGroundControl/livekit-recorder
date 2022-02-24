package egress

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/utils"
	lksdk "github.com/livekit/server-sdk-go"
)

type Service interface {
	StartRecording(ctx context.Context, req StartRecordingRequest) error
	StopRecording(ctx context.Context, req StopRecordingRequest) error
}

type service struct {
	// Info
	name string
	url  string

	// State
	lock sync.Mutex
	bots map[string]*bot

	// Services
	auth  *authProvider
	lksvc *lksdk.RoomServiceClient
}

const recordbotPrefix = "RB_"

var ErrUrlMustHaveWS = errors.New("url must contain either ws:// or wss://")

func NewService(url string, apiKey string, apiSecret string) (Service, error) {
	// By convention, we're passing ws://... in `url` , but for
	// lksdk.NewRoomServiceClient, it expects http:// . Need to check for wss too
	var tcpUrl string
	if strings.Contains(url, "ws://") {
		tcpUrl = strings.ReplaceAll(url, "ws://", "http://")
	} else if strings.Contains(url, "wss://") {
		tcpUrl = strings.ReplaceAll(url, "wss://", "https://")
	} else {
		return nil, ErrUrlMustHaveWS
	}

	// Initialise services
	auth := NewAuthProvider(apiKey, apiSecret)
	lksvc := lksdk.NewRoomServiceClient(tcpUrl, apiKey, apiSecret)

	return &service{
		name:  utils.NewGuid(recordbotPrefix),
		url:   url,
		lock:  sync.Mutex{},
		bots:  make(map[string]*bot),
		auth:  auth,
		lksvc: lksvc,
	}, nil
}

func (s *service) StartRecording(ctx context.Context, req StartRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Check if bot is already in the room; if not, create one
	_, found := s.bots[req.Room]
	if !found {
		// Create empty token
		token, err := s.auth.buildEmptyToken(req.Room, s.name)
		if err != nil {
			return err
		}

		// Create bot
		b, err := createBot(s.url, token)
		if err != nil {
			return err
		}

		// Attach the bot
		s.bots[req.Room] = b
	}
	b := s.bots[req.Room]

	// Get participant info
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}

	// For all recordable tracks of that participant,
	// construct track requests and sids
	var trackSids []string
	var trackReqs []TrackRequest
	for _, track := range s.getRecordableTracks(pi) {
		trackSids = append(trackSids, track.Sid)
		trackReqs = append(trackReqs, TrackRequest{
			SID:    track.Sid,
			Output: req.Output,
		})
	}

	// Push the track requests
	b.pushTrackRequests(trackReqs)

	// Update subscription so the bot can subscribe
	return s.updateTrackSubscriptions(ctx, req.Room, trackSids, true)
}

func (s *service) StopRecording(ctx context.Context, req StopRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Get participant info
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}

	// Get all recordable tracks
	var trackSids []string
	for _, track := range s.getRecordableTracks(pi) {
		trackSids = append(trackSids, track.Sid)
	}

	// Stop all the recorders for the participant tracks
	b := s.bots[req.Room]
	for _, trackSid := range trackSids {
		err = b.stopRecording(trackSid)
		if err != nil {
			logger.Warnw("cannot stop recorder", err, "track SID", trackSid)
		}
	}

	// Remove the subscription
	return s.updateTrackSubscriptions(ctx, req.Room, trackSids, false)
}

func (s *service) getRecordableTracks(pi *livekit.ParticipantInfo) []*livekit.TrackInfo {
	var tracks []*livekit.TrackInfo = []*livekit.TrackInfo{}
	for _, track := range pi.Tracks {
		if track.Type == livekit.TrackType_AUDIO || track.Type == livekit.TrackType_VIDEO {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func (s *service) updateTrackSubscriptions(ctx context.Context, room string, sids []string, sub bool) error {
	_, err := s.lksvc.UpdateSubscriptions(ctx, &livekit.UpdateSubscriptionsRequest{
		Room:      room,
		Identity:  s.name,
		TrackSids: sids,
		Subscribe: sub,
	})
	return err
}
