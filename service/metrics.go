package service

import (
	"github.com/prometheus/client_golang/prometheus"
)

type WSUDPCollector struct {
	wsReadCoroutineGauge  prometheus.Gauge
	wsWriteCoroutineGauge prometheus.Gauge
}

func NewWSUDPCollector() *WSUDPCollector {
	return &WSUDPCollector{
		wsReadCoroutineGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "current_ws_read_coroutines",
				Help: "The number of goroutines that are currently reading from the WebSocket",
			},
		),
		wsWriteCoroutineGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "current_ws_write_coroutines",
				Help: "The number of goroutines that are currently writing to the WebSocket",
			},
		),
	}
}

func (c *WSUDPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.wsReadCoroutineGauge.Desc()
	ch <- c.wsWriteCoroutineGauge.Desc()
}

func (c *WSUDPCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.wsReadCoroutineGauge
	ch <- c.wsWriteCoroutineGauge
}
