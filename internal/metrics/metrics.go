package metrics

import (
  "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
  ServerHits  prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ServerHits: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "server_hits_total",
			Help: "The total number of requests to the server",
		}),
	}
	return m
}

