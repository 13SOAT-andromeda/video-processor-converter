package metrics

import (
	"log/slog"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
)

var _ ports.Metrics = (*DogStatsD)(nil)

type DogStatsD struct {
	client *statsd.Client
	log    *slog.Logger
}

func New(addr string, log *slog.Logger) (*DogStatsD, error) {
	client, err := statsd.New(addr, statsd.WithNamespace("video.processor.converter."))
	if err != nil {
		return nil, err
	}
	return &DogStatsD{client: client, log: log}, nil
}

func (d *DogStatsD) Count(name string, value int64, tags ...string) {
	if err := d.client.Count(name, value, tags, 1); err != nil {
		d.log.Warn("metrics count failed", "metric", name, "err", err)
	}
}

func (d *DogStatsD) Timing(name string, dur time.Duration, tags ...string) {
	if err := d.client.Distribution(name, float64(dur.Milliseconds()), tags, 1); err != nil {
		d.log.Warn("metrics timing failed", "metric", name, "err", err)
	}
}

func (d *DogStatsD) Distribution(name string, value float64, tags ...string) {
	if err := d.client.Distribution(name, value, tags, 1); err != nil {
		d.log.Warn("metrics distribution failed", "metric", name, "err", err)
	}
}
