package api

import (
	"strconv"
	"time"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

var (
	includeOrgLabel = os.Getenv("AURA_METRICS_LABELS_ORG") == "true"
	reqDuration     = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "aura",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"method", "path", "status"},
	)
	decisionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "verify_decisions_total", Help: "Verification decisions by outcome (optionally labeled by org)"},
		[]string{"decision", "org"},
	)
)

func init() {
	prometheus.MustRegister(reqDuration, reqTotal, decisionTotal)
}

// MetricsMiddleware records basic HTTP metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Seconds()
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		labels := []string{c.Request.Method, path, toStr(status)}
		observer := reqDuration.WithLabelValues(labels...)
		// attach exemplar with trace_id if present
		if sc := trace.SpanContextFromContext(c.Request.Context()); sc.IsValid() {
			if eo, ok := observer.(prometheus.ExemplarObserver); ok {
				eo.ObserveWithExemplar(dur, prometheus.Labels{"trace_id": sc.TraceID().String()})
			} else {
				observer.Observe(dur)
			}
		} else {
			observer.Observe(dur)
		}
		reqTotal.With(prometheus.Labels{"method": c.Request.Method, "path": path, "status": toStr(status)}).Inc()
	}
}

func toStr(i int) string { return strconv.Itoa(i) }

// RecordDecision increments a decision counter
func RecordDecision(dec string, org string) {
	if !includeOrgLabel {
		org = ""
	}
	decisionTotal.With(prometheus.Labels{"decision": dec, "org": org}).Inc()
}
