package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
)

var _ ports.FrameExtractor = (*Extractor)(nil)

type Extractor struct {
	binPath   string // "ffmpeg"
	frameRate int    // 1
}

func NewExtractor(frameRate int) *Extractor {
	return &Extractor{binPath: "ffmpeg", frameRate: frameRate}
}

func (e *Extractor) ExtractFrames(ctx context.Context, inputPath, outDir string) (int, error) {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return 0, fmt.Errorf("mkdir frames: %w", err)
	}
	pattern := filepath.Join(outDir, "frame_%04d.jpg")

	cmd := exec.CommandContext(ctx, e.binPath,
		"-i", inputPath,
		"-vf", "fps="+strconv.Itoa(e.frameRate),
		"-q:v", "2",
		"-y",
		pattern,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("ffmpeg exec: %w (output: %s)", err, string(out))
	}
	frames, err := filepath.Glob(filepath.Join(outDir, "*.jpg"))
	if err != nil {
		return 0, fmt.Errorf("glob frames: %w", err)
	}
	if len(frames) == 0 {
		return 0, fmt.Errorf("ffmpeg produced no frames")
	}
	return len(frames), nil
}
