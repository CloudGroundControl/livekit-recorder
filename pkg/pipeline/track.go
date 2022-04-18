package pipeline

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/lithammer/shortuuid/v4"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
	"time"
)

const RecordingsDir = "recordings"

var gstInitialised = false

type Pipeline interface {
	Start()
	Stop()
	Cleanup()
}

type trackPipeline struct {
	track  *webrtc.TrackRemote
	ctx    context.Context
	cancel context.CancelFunc

	pipeline *gst.Pipeline
	src      *app.Source
	elements []*gst.Element

	runtime    time.Time
	runtimeSet bool
}

func NewTrackPipeline(track *webrtc.TrackRemote) (Pipeline, error) {
	var err error
	if !gstInitialised {
		gst.Init(nil)
		gstInitialised = true
	}

	pipe, err := gst.NewPipeline("track")
	if err != nil {
		return nil, err
	}

	// App source
	srcElement, err := gst.NewElement("appsrc")
	if err != nil {
		return nil, err
	}
	src := app.SrcFromElement(srcElement)

	// Depay & mux
	var depay *gst.Element
	switch track.Codec().MimeType {
	case webrtc.MimeTypeVP8:
		depay, err = gst.NewElement("rtpvp8depay")
	case webrtc.MimeTypeH264:
		depay, err = gst.NewElement("rtph264depay")
	case webrtc.MimeTypeOpus:
		depay, err = gst.NewElement("rtpopusdepay")
	}
	if err != nil {
		return nil, err
	}

	// Queue
	queue, err := gst.NewElement("queue")
	if err != nil {
		return nil, err
	}

	// Mux
	var mux *gst.Element
	var ext string
	switch track.Codec().MimeType {
	case webrtc.MimeTypeVP8:
		mux, err = gst.NewElement("webmmux")
		ext = "webm"
	case webrtc.MimeTypeH264:
		mux, err = gst.NewElement("mp4mux")
		ext = "mp4"
	case webrtc.MimeTypeOpus:
		mux, err = gst.NewElement("oggmux")
		ext = "ogg"
	}
	if err != nil {
		return nil, err
	}

	// Filesink
	filesink, err := gst.NewElement("filesink")
	if err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s/%s.%s", RecordingsDir, shortuuid.New(), ext)
	if err = filesink.SetProperty("location", filename); err != nil {
		return nil, err
	}
	if err = filesink.SetProperty("sync", false); err != nil {
		return nil, err
	}

	// Add pipeline elements
	elements := []*gst.Element{src.Element, depay, queue, mux, filesink}
	err = pipe.AddMany(elements...)
	if err != nil {
		return nil, err
	}

	// Link elements
	err = gst.ElementLinkMany(elements...)
	if err != nil {
		return nil, err
	}

	// Link queue static pad and mux request pad
	var muxPad *gst.Pad
	switch track.Kind() {
	case webrtc.RTPCodecTypeAudio:
		muxPad = mux.GetRequestPad("audio_%u")
	case webrtc.RTPCodecTypeVideo:
		muxPad = mux.GetRequestPad("video_%u")
	default:
		return nil, fmt.Errorf("track kind not supported: %s", track.Kind().String())
	}
	if ret := queue.GetStaticPad("src").Link(muxPad); ret != gst.PadLinkOK {
		return nil, fmt.Errorf("cannot link pad: %s", ret.String())
	}

	p := &trackPipeline{
		track:      track,
		pipeline:   pipe,
		src:        src,
		elements:   elements,
		runtimeSet: false,
	}

	return p, nil
}

func (p *trackPipeline) Start() {
	p.ctx, p.cancel = context.WithCancel(context.TODO())
	p.start()
	if err := p.pipeline.Start(); err != nil {
		log.Errorf("error starting recording: %s", err)
	}
}

func (p *trackPipeline) start() {
	var err error
	defer func() {
		// Log any errors
		if err != nil {
			log.Errorf("error stopping recording: %s", err)
		}

		// Send EOS from app source
		ret := p.src.EndStream()
		if ret != gst.FlowOK {
			log.Errorf("error sending EOS: %s", ret.String())
		}
	}()

	var pkt *rtp.Packet
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			pkt, _, err = p.track.ReadRTP()
			if err != nil {
				return
			}

			err = p.push(pkt)
			if err != nil {
				return
			}
		}
	}
}

func (p *trackPipeline) push(pkt *rtp.Packet) error {
	// Initialise buffer
	b := gst.NewBufferFromBytes(pkt.Payload)
	defer b.Unref()

	// Set PTS
	ts := time.Unix(0, int64(pkt.Timestamp))
	d := p.runtime.Sub(ts)
	b.SetPresentationTimestamp(d)

	// Push buffer
	ret := p.src.PushBuffer(b)
	if ret != gst.FlowOK {
		return errors.New(ret.String())
	}
	return nil
}

func (p *trackPipeline) Stop() {
	// Stop recorder goroutine
	p.cancel()
}

func (p *trackPipeline) Cleanup() {
	// Set state to null and stop all activity
	err := p.pipeline.SetState(gst.StateNull)
	if err != nil {
		log.Error("cannot set state to null")
	}

	// Unref elements
	for _, el := range p.elements {
		el.Unref()
	}
	p.pipeline.Unref()
}
