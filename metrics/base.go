package metrics

import (
	"github.com/evolutionlandorg/block-scan/util"
	"github.com/evolutionlandorg/block-scan/util/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cast"
	"net/http"
	"sync"
)

var (
	once    sync.Once
	metrics Metrics
	// enableMetrics 是否开启metrics
	enableMetrics = cast.ToBool(util.GetEnv("ENABLE_METRICS", "false"))
	port          = util.GetEnv("METRICS_PORT", ":8085")
)

func init() {
	if enableMetrics {
		once.Do(func() {
			go func() {
				metrics = newPrometheusMetrics()
				http.Handle("/metrics", promhttp.Handler())
				log.Info("block-scan metrics server start", "port", port)
				util.Panic(http.ListenAndServe(port, nil))
			}()
		})
	}
	metrics = newFakeMetrics()
}

type Metrics interface {
	ScanTxTotal(network string, value ...float64)
	ScanCallbackTotal(network string, value ...float64)
}

func NewMetrics() Metrics {
	return metrics
}
