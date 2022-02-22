package egress

import (
	"context"
	"strings"
	"sync"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
)

type Service interface {
	StartRecording(ctx context.Context, req StartRecordingRequest) error
	StopRecording(ctx context.Context, req StopRecordingRequest) error
}

type service struct {
	url   string
	auth  authProvider
	lock  sync.Mutex
	bots  map[string]*bot
	name  string
	lksvc *lksdk.RoomServiceClient
}

func NewService(url string, apiKey string, apiSecret string) Service {
	auth := authProvider{APIKey: apiKey, APISecret: apiSecret}
	httpUrl := strings.ReplaceAll(url, "ws", "http")
	return &service{
		url:   url,
		auth:  auth,
		lock:  sync.Mutex{},
		bots:  make(map[string]*bot),
		name:  "egress",
		lksvc: lksdk.NewRoomServiceClient(httpUrl, apiKey, apiSecret),
	}
}

func (s *service) StartRecording(ctx context.Context, req StartRecordingRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Check if bot is already in the room; if not, create one and join
	var b *bot
	b, ok := s.bots[req.Room]
	if !ok {
		// Create empty token
		token, err := s.auth.buildEmptyToken(req.Room, s.name)
		if err != nil {
			return err
		}

		// Create bot and attach
		b, err = createBot(s.url, token)
		if err != nil {
			return err
		}
		s.bots[req.Room] = b
	}

	// Get participant info
	rp, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}

	// Filter out the participant tracks according to the requested OutputChannel
	tracks := s.filterParticipantTracks(rp.Tracks, req.Channel)

	// Push track request and SID based on the filtered tracks
	var trackSids []string
	for _, track := range tracks {
		b.pushTrackRequest(TrackRequest{
			SID:    track.Sid,
			Output: req.File,
		})
		trackSids = append(trackSids, track.Sid)
	}

	// Update subscription so the bot can subscribe
	return s.updateTrackSubscriptions(ctx, req.Room, trackSids, true)
}

func (s *service) filterParticipantTracks(tracks []*livekit.TrackInfo, channel OutputChannel) []*livekit.TrackInfo {
	// Build a map to categorise the track by their type for easier access later on
	tracksByType := make(map[livekit.TrackType]*livekit.TrackInfo)
	for _, track := range tracks {
		tracksByType[track.Type] = track
	}

	// Select which tracks to output based on requested channel
	switch channel {
	case OutputChannelAV:
		return []*livekit.TrackInfo{
			tracksByType[livekit.TrackType_VIDEO],
			tracksByType[livekit.TrackType_AUDIO],
		}
	case OutputChannelAudio:
		return []*livekit.TrackInfo{
			tracksByType[livekit.TrackType_AUDIO],
		}
	case OutputChannelVideo:
		return []*livekit.TrackInfo{
			tracksByType[livekit.TrackType_VIDEO],
		}
	}

	// Return empty list by default
	return []*livekit.TrackInfo{}
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

func (s *service) StopRecording(ctx context.Context, req StopRecordingRequest) error {
	// Get participant info
	rp, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.Room,
		Identity: req.Participant,
	})
	if err != nil {
		return err
	}

	// Get all tracks of participant without filtering of requested OutputChannel
	var trackSids []string
	for _, track := range rp.Tracks {
		trackSids = append(trackSids, track.Sid)
	}

	// Just need to update subscription
	return s.updateTrackSubscriptions(ctx, req.Room, trackSids, false)
}
