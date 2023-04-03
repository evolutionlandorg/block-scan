package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	scanTxTotal       *prometheus.CounterVec
	scanCallbackTotal *prometheus.CounterVec
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

func newPrometheusMetrics() *PrometheusMetrics {
	var labelNames = []string{
		"network",
	}
	l := &PrometheusMetrics{
		scanTxTotal:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "scan_tx_total", Help: "The total number of scan tx"}, labelNames),
		scanCallbackTotal: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "scan_callback_total", Help: "The total number of scan callback"}, labelNames),
	}
	prometheus.MustRegister(l.scanTxTotal, l.scanCallbackTotal)
	return l
}
