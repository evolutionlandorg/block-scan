package metrics

import (
	"github.com/evolutionlandorg/block-scan/util"
	"github.com/evolutionlandorg/block-scan/util/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Type string

var (
	PrometheusType Type = "prometheus"
	FakeType       Type = "fake"
)

type Metrics interface {
	ScanTxTotal(network string, value ...float64)
	ScanCallbackTotal(network string, value ...float64)
	ScanCallbackErrorTotal(network string, value ...float64)
}

func NewMetrics(metricsType Type) Metrics {
	switch metricsType {
	case PrometheusType:
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			port := util.GetEnv("METRICS_PORT", ":8085")
			log.Info("block-scan metrics server start", "port", port)
			util.Panic(http.ListenAndServe(port, nil))
		}()
		return newPrometheusMetrics()
	case FakeType:
		return newFakeMetrics()
	default:
		return newFakeMetrics()
	}
}
