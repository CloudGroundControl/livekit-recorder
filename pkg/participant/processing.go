package participant

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/labstack/gommon/log"
	"github.com/lithammer/shortuuid/v4"
)

func (p *participant) process() error {
	// If there is no video, don't containerise
	if p.vr == nil {
		p.data.Output = p.af

		// Check if we want to upload the audio file
		if p.uploader != nil {
			p.data.Output = fmt.Sprintf("%s/%s", p.uploader.GetDirectory(), p.data.Output)
			go func() {
				err := p.upload(p.af)
				if err != nil {
					log.Errorf("cannot upload audio | error: %v, output: %s, participant: %s", err, p.data.Output, p.data.Identity)
				}
				log.Infof("uploaded audio | output: %s, participant: %s", p.data.Output, p.data.Identity)
			}()
		}
		return nil
	}

	// Containerise file
	filename, err := p.containerise()
	if err != nil {
		return err
	}
	p.data.Output = filename
	log.Debugf("containerised file | output: %s, participant: %s, video: %s, audio: %s", p.data.Output, p.data.Identity, p.vf, p.af)

	// If there are no errors during containerisation, delete the raw media files
	if p.vf != "" {
		if err = os.Remove(p.vf); err != nil {
			return err
		}
		log.Debugf("removed raw video | file: %s", p.vf)
	}
	if p.af != "" {
		if err = os.Remove(p.af); err != nil {
			return err
		}
		log.Debugf("removed raw audio | file: %s", p.af)
	}

	// Check if we want to upload the container file
	if p.uploader != nil {
		p.data.Output = fmt.Sprintf("%s/%s", p.uploader.GetDirectory(), p.data.Output)
		go func() {
			err := p.upload(filename)
			if err != nil {
				log.Errorf("cannot upload container | error: %v, output: %s, participant: %s", err, p.data.Output, p.data.Identity)
			}
			log.Infof("uploaded container | output: %s, participant: %s", p.data.Output, p.data.Identity)
		}()
	}
	return nil
}

func (p *participant) containerise() (string, error) {
	// We have 4 cases:
	// 1. Video = IVF, Audio = nil. Containerise as webm
	// 2. Video = H264, Audio = nil. Containerise as mp4
	// 3. Video = IVF, Audio = OGG. Containerise as webm with 2 file inputs
	// 4. Video = H264, Audio = OGG. Containerise as mp4 with 2 file inputs + extra flag for OGG

	var (
		videoExt recorder.MediaExtension = ""
		audioExt recorder.MediaExtension = ""
		cmd      *exec.Cmd
		filename string
	)

	if p.vt != nil {
		videoExt = recorder.GetMediaExtension(p.vt.Codec().MimeType)
	}
	if p.at != nil {
		audioExt = recorder.GetMediaExtension(p.at.Codec().MimeType)
	}

	// Generate file ID
	fileID := fmt.Sprintf("%s/%s", RecordingsDir, shortuuid.New())

	// Case 1
	if videoExt == recorder.MediaIVF && audioExt == "" {
		filename = fmt.Sprintf("%s.%s", fileID, "webm")
		cmd = exec.Command("ffmpeg",
			"-i", p.vf,
			"-c:v", "copy",
			"-loglevel", "error",
			"-y", filename)
	}

	// Case 2
	if videoExt == recorder.MediaH264 && audioExt == "" {
		filename = fmt.Sprintf("%s.%s", fileID, "mp4")
		cmd = exec.Command("ffmpeg",
			"-i", p.vf,
			"-c:v", "copy",
			"-loglevel", "error",
			"-y", filename)
	}

	// Case 3
	if videoExt == recorder.MediaIVF && audioExt == recorder.MediaOGG {
		filename = fmt.Sprintf("%s.%s", fileID, "webm")
		cmd = exec.Command("ffmpeg",
			"-i", p.vf,
			"-i", p.af,
			"-c:v", "copy",
			"-c:a", "copy",
			"-loglevel", "error",
			"-y", "-shortest", filename)
	}

	// Case 4
	if videoExt == recorder.MediaH264 && audioExt == recorder.MediaOGG {
		filename = fmt.Sprintf("%s.%s", fileID, "mp4")
		cmd = exec.Command("ffmpeg",
			"-i", p.vf,
			"-i", p.af,
			"-c:v", "copy",
			"-c:a", "copy",
			"-loglevel", "error",
			"-y", "-shortest", filename)
	}

	// Execute command
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	return filename, err
}

func (p *participant) upload(filename string) error {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	// Try uploading
	key := strings.ReplaceAll(filename, RecordingsDir+"/", "")
	err = p.uploader.Upload(key, file)
	if err != nil {
		return err
	}

	// If there are no errors after uploading, delete the file
	return os.Remove(filename)
}
