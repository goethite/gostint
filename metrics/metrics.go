package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// inspried by https://github.com/766b/chi-prometheus

const (
	durationName = "gostint_request_duration_seconds"
	durationDesc = "gostint request duration in seconds"
)

type Metrics struct {
	duration *prometheus.HistogramVec
}

func NewMetrics(name string) func(next http.Handler) http.Handler {
	var m Metrics
	m.duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        durationName,
			Help:        durationDesc,
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{
			"code",
			"method",
			"path",
		},
	)
	return m.handler
}

func (m Metrics) handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		m.duration.WithLabelValues(
			fmt.Sprintf("%d", ww.Status()),
			r.Method,
			r.URL.Path,
		).Observe(
			float64(time.Since(start).Seconds()),
		)
	}
	return http.HandlerFunc(fn)
}
