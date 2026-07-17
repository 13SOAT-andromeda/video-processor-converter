package ffprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

var _ ports.VideoProber = (*Prober)(nil)

type Prober struct{ binPath string } // "ffprobe" (PATH) por padrão

func NewProber() *Prober { return &Prober{binPath: "ffprobe"} }

type probeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func parseProbeOutput(out []byte) (domain.Resolution, error) {
	var parsed probeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return domain.Resolution{}, fmt.Errorf("ffprobe parse: %w", err)
	}
	if len(parsed.Streams) == 0 {
		return domain.Resolution{}, fmt.Errorf("ffprobe: no video stream")
	}
	return domain.Resolution{Width: parsed.Streams[0].Width, Height: parsed.Streams[0].Height}, nil
}

func (p *Prober) Probe(ctx context.Context, path string) (domain.Resolution, error) {
	cmd := exec.CommandContext(ctx, p.binPath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return domain.Resolution{}, fmt.Errorf("ffprobe exec: %w", err)
	}
	return parseProbeOutput(out)
}
