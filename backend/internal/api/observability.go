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
	// External ops (e.g., docker codegen, SMTP)
	externalDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: "aura", Name: "external_op_duration_seconds", Help: "Duration of external operations"},
		[]string{"op", "outcome"},
	)
	externalTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "external_op_total", Help: "Total external operations"},
		[]string{"op", "outcome"},
	)
	breakerOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "aura", Name: "circuit_breaker_open", Help: "Circuit breaker state: 1=open, 0=closed"},
		[]string{"breaker"},
	)
	// DLQ metrics
	dlqInsertTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "dlq_insert_total", Help: "Total DLQ insertions"},
		[]string{"stream", "reason"},
	)
	dlqDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "aura", Name: "dlq_depth", Help: "Current DLQ depth"},
		[]string{"stream"},
	)
	// Queue pending (consumer group pending messages)
	queuePending = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "aura", Name: "queue_pending", Help: "Pending messages in queue consumer groups"},
		[]string{"stream"},
	)
	// Verification reason taxonomy
	decisionReasonTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "verify_reason_total", Help: "Verification counts by reason and decision"},
		[]string{"reason", "decision", "org"},
	)
	// API key usage
	apiKeyUsageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "api_key_usage_total", Help: "API key usage by key prefix (and optional org)"},
		[]string{"key_prefix", "org"},
	)
	// Cache metrics (RED-compatible)
	cacheHitTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "cache_hit_total", Help: "Cache hits by component and key"},
		[]string{"component", "key"},
	)
	cacheMissTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "cache_miss_total", Help: "Cache misses by component and key"},
		[]string{"component", "key"},
	)
	// Trust token issuance metrics
	trustTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "aura", Name: "trust_tokens_total", Help: "Trust tokens built by source/alg and outcome"},
		[]string{"source", "alg", "outcome", "org"},
	)
	// In-flight verification gauge and quick error counter
	verifyInflight = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: "aura", Name: "verify_inflight", Help: "Current number of in-flight verify requests"},
	)
	verifyQuickRejectTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: "aura", Name: "verify_quick_reject_total", Help: "Total verify requests rejected immediately due to backpressure"},
	)
)

func init() {
	prometheus.MustRegister(reqDuration, reqTotal, decisionTotal, externalDuration, externalTotal, breakerOpen, dlqInsertTotal, dlqDepth, queuePending, decisionReasonTotal, apiKeyUsageTotal, cacheHitTotal, cacheMissTotal, trustTokensTotal, verifyInflight, verifyQuickRejectTotal)
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

// RecordDecisionReason increments counters for reason taxonomy
func RecordDecisionReason(reason, decision, org string) {
	if !includeOrgLabel {
		org = ""
	}
	if reason == "" {
		reason = "unspecified"
	}
	decisionReasonTotal.With(prometheus.Labels{"reason": reason, "decision": decision, "org": org}).Inc()
}

// RecordExternalOp records an external operation metric with duration and outcome
func RecordExternalOp(op string, dur time.Duration, success bool) {
	outcome := "success"
	if !success {
		outcome = "error"
	}
	externalDuration.WithLabelValues(op, outcome).Observe(dur.Seconds())
	externalTotal.WithLabelValues(op, outcome).Inc()
}

// SetBreakerState updates the breaker state gauge (1=open, 0=closed)
func SetBreakerState(name string, open bool) {
	if open {
		breakerOpen.WithLabelValues(name).Set(1)
	} else {
		breakerOpen.WithLabelValues(name).Set(0)
	}
}

// RecordDLQInsert increments the DLQ insertion counter
func RecordDLQInsert(stream string, reason string) {
	dlqInsertTotal.WithLabelValues(stream, reason).Inc()
}

// SetDLQDepth sets the current DLQ depth gauge
func SetDLQDepth(stream string, n int64) {
	dlqDepth.WithLabelValues(stream).Set(float64(n))
}

// SetQueuePending sets the current pending messages gauge
func SetQueuePending(stream string, n int64) {
	queuePending.WithLabelValues(stream).Set(float64(n))
}

// RecordAPIKeyUsage increments usage counters labeled by key prefix (and org if enabled)
func RecordAPIKeyUsage(keyPrefix, org string) {
	if !includeOrgLabel {
		org = ""
	}
	apiKeyUsageTotal.With(prometheus.Labels{"key_prefix": keyPrefix, "org": org}).Inc()
}

// RecordCacheHit increments the cache hit counter for a component/key
func RecordCacheHit(component, key string) { cacheHitTotal.WithLabelValues(component, key).Inc() }

// RecordCacheMiss increments the cache miss counter for a component/key
func RecordCacheMiss(component, key string) { cacheMissTotal.WithLabelValues(component, key).Inc() }

// RecordTrustToken records a trust token issuance attempt with outcome
func RecordTrustToken(source, alg string, success bool, org string) {
	if !includeOrgLabel {
		org = ""
	}
	outcome := "success"
	if !success {
		outcome = "error"
	}
	trustTokensTotal.With(prometheus.Labels{"source": source, "alg": alg, "outcome": outcome, "org": org}).Inc()
}

// SetVerifyInflight sets the current in-flight verify gauge
func SetVerifyInflight(n int) { verifyInflight.Set(float64(n)) }

// IncVerifyQuickReject increments the quick-reject counter
func IncVerifyQuickReject() { verifyQuickRejectTotal.Inc() }
