package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	scanTxTotal            *prometheus.CounterVec
	scanCallbackErrorTotal *prometheus.CounterVec
	scanCallbackTotal      *prometheus.CounterVec
}

func (p PrometheusMetrics) ScanTxTotal(network string, value ...float64) {
	var v = 1.0
	if len(value) > 0 {
		v = value[0]
	}
	p.scanTxTotal.With(prometheus.Labels{"network": network}).Add(v)
}

func (p PrometheusMetrics) ScanCallbackTotal(network string, value ...float64) {
	var v = 1.0
	if len(value) > 0 {
		v = value[0]
	}
	p.scanCallbackTotal.With(prometheus.Labels{"network": network}).Add(v)
}

func (p PrometheusMetrics) ScanCallbackErrorTotal(network string, value ...float64) {
	var v = 1.0
	if len(value) > 0 {
		v = value[0]
	}
	p.scanCallbackErrorTotal.With(prometheus.Labels{"network": network}).Add(v)
}

func newPrometheusMetrics() *PrometheusMetrics {
	var labelNames = []string{
		"network",
	}
	return &PrometheusMetrics{
		scanTxTotal:            prometheus.NewCounterVec(prometheus.CounterOpts{Name: "scan_tx_total", Help: "The total number of scan tx"}, labelNames),
		scanCallbackTotal:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "scan_callback_total", Help: "The total number of scan callback"}, labelNames),
		scanCallbackErrorTotal: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "scan_callback_error_total", Help: "The total number of scan callback error"}, labelNames),
	}
}
