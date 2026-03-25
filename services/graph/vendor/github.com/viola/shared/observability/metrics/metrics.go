package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Common labels used across all services
const (
	LabelTenantID  = "tenant_id"
	LabelService   = "service"
	LabelRule      = "rule"
	LabelSeverity  = "severity"
	LabelStatus    = "status"
	LabelEndpoint  = "endpoint"
	LabelMethod    = "method"
	LabelEventType = "event_type"
)

// Registry holds all Prometheus metrics for a service
type Registry struct {
	namespace string

	// Event processing metrics
	EventsProcessed  *prometheus.CounterVec
	EventsErrors     *prometheus.CounterVec
	ProcessingTime   *prometheus.HistogramVec

	// Alert metrics
	AlertsGenerated  *prometheus.CounterVec
	AlertsSuppressed *prometheus.CounterVec
	AlertRiskScore   *prometheus.HistogramVec

	// Graph metrics
	GraphNodes       *prometheus.GaugeVec
	GraphEdges       *prometheus.GaugeVec
	GraphCleanups    *prometheus.CounterVec
	RiskScoreTime    *prometheus.HistogramVec

	// API metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
	HTTPErrors       *prometheus.CounterVec

	// Consumer metrics
	ConsumerLag      *prometheus.GaugeVec
	ConsumerOffset   *prometheus.GaugeVec
}

// NewRegistry creates a new metrics registry for a service
func NewRegistry(serviceName string) *Registry {
	return &Registry{
		namespace: serviceName,

		// Event processing
		EventsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "events_processed_total",
				Help:      "Total number of events processed",
			},
			[]string{LabelTenantID, LabelEventType},
		),

		EventsErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "events_errors_total",
				Help:      "Total number of event processing errors",
			},
			[]string{LabelTenantID, LabelEventType},
		),

		ProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "event_processing_duration_seconds",
				Help:      "Time spent processing events",
				Buckets:   prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{LabelTenantID, LabelEventType},
		),

		// Alerts
		AlertsGenerated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "alerts_generated_total",
				Help:      "Total number of alerts generated",
			},
			[]string{LabelTenantID, LabelRule, LabelSeverity},
		),

		AlertsSuppressed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "alerts_suppressed_total",
				Help:      "Total number of alerts suppressed by graph context",
			},
			[]string{LabelTenantID, LabelRule},
		),

		AlertRiskScore: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "alert_risk_score",
				Help:      "Distribution of alert risk scores",
				Buckets:   []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			},
			[]string{LabelTenantID, LabelSeverity},
		),

		// Graph
		GraphNodes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: serviceName,
				Name:      "graph_nodes",
				Help:      "Current number of nodes in the graph",
			},
			[]string{LabelTenantID},
		),

		GraphEdges: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: serviceName,
				Name:      "graph_edges",
				Help:      "Current number of edges in the graph",
			},
			[]string{LabelTenantID},
		),

		GraphCleanups: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "graph_cleanups_total",
				Help:      "Total number of expired edges cleaned up",
			},
			[]string{LabelTenantID},
		),

		RiskScoreTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "risk_score_computation_duration_seconds",
				Help:      "Time spent computing risk scores",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{LabelTenantID},
		),

		// HTTP API
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{LabelEndpoint, LabelMethod, LabelStatus},
		),

		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latencies",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{LabelEndpoint, LabelMethod},
		),

		HTTPErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "http_errors_total",
				Help:      "Total number of HTTP errors",
			},
			[]string{LabelEndpoint, LabelMethod, LabelStatus},
		),

		// Consumer lag
		ConsumerLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: serviceName,
				Name:      "consumer_lag",
				Help:      "Current consumer lag (messages behind)",
			},
			[]string{"topic", "partition"},
		),

		ConsumerOffset: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: serviceName,
				Name:      "consumer_offset",
				Help:      "Current consumer offset",
			},
			[]string{"topic", "partition"},
		),
	}
}

// Handler returns the Prometheus HTTP handler for scraping
func (r *Registry) Handler() http.Handler {
	return promhttp.Handler()
}

// SuppressionRate computes the percentage of alerts suppressed
// Call this periodically to log/expose suppression rate
func SuppressionRate(generated, suppressed float64) float64 {
	if generated == 0 {
		return 0
	}
	return (suppressed / generated) * 100
}
