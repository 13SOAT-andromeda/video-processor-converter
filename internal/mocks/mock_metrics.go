package mocks

import "time"

type MetricCall struct {
	Name  string
	Value float64
	Tags  []string
}

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
