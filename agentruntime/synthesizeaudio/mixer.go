package synthesizeaudio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"hanhan-radio/agentruntime/log"
)

// Concat concatenates a voice intro (WAV) with a music file using ffmpeg.
// voicePath and musicPath are the input files; outputPath is the destination.
func Concat(ctx context.Context, voicePath, musicPath, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	args := []string{
		"-y",
		"-i", voicePath,
		"-i", musicPath,
		"-filter_complex",
		"[0:a]aformat=channel_layouts=stereo:sample_rates=44100[voice];[voice][1:a]concat=n=2:v=0:a=1",
		"-b:a", "192k",
		"-c:a", "libmp3lame",
		outputPath,
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.L().Errorw("ffmpeg failed", "stderr", stderr.String())
		return fmt.Errorf("ffmpeg concat: %w\n%s", err, stderr.String())
	}

	return nil
}

// ProbeDuration returns the duration of an audio file in seconds using ffprobe.
func ProbeDuration(ctx context.Context, filePath string) (int, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w", err)
	}

	var seconds float64
	if _, err := fmt.Sscanf(string(out), "%f", &seconds); err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", string(out), err)
	}

	return int(seconds), nil
}
