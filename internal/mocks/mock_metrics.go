package mocks

import "time"

// MetricCall registra uma chamada individual a MockMetrics.
type MetricCall struct {
	Name  string
	Value float64
	Tags  []string
}

// MockMetrics é um stub no-op de ports.Metrics: não exige expectations (.On)
// para não sobrecarregar testes que não verificam métricas, mas guarda cada
// chamada para os poucos testes que quiserem checar o que foi emitido.
type MockMetrics struct {
	Counts        []MetricCall
	Timings       []MetricCall
	Distributions []MetricCall
}

func (m *MockMetrics) Count(name string, value int64, tags ...string) {
	m.Counts = append(m.Counts, MetricCall{Name: name, Value: float64(value), Tags: tags})
}

func (m *MockMetrics) Timing(name string, d time.Duration, tags ...string) {
	m.Timings = append(m.Timings, MetricCall{Name: name, Value: float64(d), Tags: tags})
}

func (m *MockMetrics) Distribution(name string, value float64, tags ...string) {
	m.Distributions = append(m.Distributions, MetricCall{Name: name, Value: value, Tags: tags})
}
