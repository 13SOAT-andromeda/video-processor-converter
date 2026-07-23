package ports

import "time"

// Metrics é o client de métricas custom (contadores, timings, distribuições).
type Metrics interface {
	Count(name string, value int64, tags ...string)
	Timing(name string, d time.Duration, tags ...string)
	Distribution(name string, value float64, tags ...string)
}
