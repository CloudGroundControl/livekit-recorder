package recorder

import (
	"os"
	"os/exec"
)

func putVideoInContainer(input string, output string) error {
	cmd := exec.Command("ffmpeg",
		"-i", input,
		"-c:v", "copy",
		"-loglevel", "error", "-y",
		output,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
