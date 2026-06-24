package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
  ServerHits  	*prometheus.CounterVec
  Latency	*prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ServerHits: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_hits_total",
				Help: "The total number of requests to the server",
			},
			[]string{"path"},
		),
		Latency: promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Help:    "Response latencies for HTTP requests.",
				Buckets: []float64{0.1, 0.3, 0.5, 1.0, 2.5, 5.0},
			},
			[]string{"path"},
		),
	}
	return m
}

