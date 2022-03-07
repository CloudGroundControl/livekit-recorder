package recording

import (
	"context"

	"github.com/livekit/protocol/livekit"
)

func (s *service) getProfileTracks(req MediaProfile, tracks []*livekit.TrackInfo) []*livekit.TrackInfo {
	var recordables []*livekit.TrackInfo
	for _, t := range tracks {
		if t.Type == livekit.TrackType_VIDEO && (req == MediaVideoOnly || req == MediaMuxedAV) {
			recordables = append(recordables, t)
		}
		if t.Type == livekit.TrackType_AUDIO && (req == MediaAudioOnly || req == MediaMuxedAV) {
			recordables = append(recordables, t)
		}
	}
	return recordables
}

type UpdateTrackSubscriptionsRequest struct {
	Room     string
	Identity string
	SIDs     []string
	Subcribe bool
}

func (s *service) updateTrackSubscriptions(ctx context.Context, req UpdateTrackSubscriptionsRequest) error {
	_, err := s.lksvc.UpdateSubscriptions(ctx, &livekit.UpdateSubscriptionsRequest{
		Room:      req.Room,
		Identity:  req.Identity,
		TrackSids: req.SIDs,
		Subscribe: req.Subcribe,
	})
	return err
}

func (s *service) RemoveBotSubscriptionCallback(bot string, room string, participant string, req MediaProfile) error {
	// Get participant info to be able to get the tracks the bot is subscribed to
	ctx := context.TODO()
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     room,
		Identity: participant,
	})
	if err != nil {
		return err
	}

	// Get the SIDs that the bot is subscribed to
	var sids []string
	for _, r := range s.getProfileTracks(req, pi.Tracks) {
		sids = append(sids, r.Sid)
	}

	// Unsubscribe the bot from the participant tracks
	return s.updateTrackSubscriptions(ctx, UpdateTrackSubscriptionsRequest{
		Room:     room,
		Identity: bot,
		SIDs:     sids,
		Subcribe: false,
	})
}

func (s *service) SuggestMediaProfile(ctx context.Context, room string, identity string) (MediaProfile, error) {
	pi, err := s.lksvc.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     room,
		Identity: identity,
	})
	if err != nil {
		return "", err
	}

	var videoEnabled, audioEnabled bool
	for _, t := range pi.Tracks {
		switch t.Type {
		case livekit.TrackType_VIDEO:
			videoEnabled = true
		case livekit.TrackType_AUDIO:
			audioEnabled = true
		}
	}

	if videoEnabled && audioEnabled {
		return MediaMuxedAV, nil
	} else if videoEnabled && !audioEnabled {
		return MediaVideoOnly, nil
	} else if audioEnabled && !videoEnabled {
		return MediaAudioOnly, nil
	}

	return "", ErrUnknownMediaProfile
}
