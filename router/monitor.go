package router

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/benitogf/katamari"
)

func monitor(server *katamari.Server) {
	subscriptions := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "active_subscriptions",
		Help: "active subscriptions",
	})
	prometheus.MustRegister(subscriptions)
	server.OnSubscribe = func(key string) error {
		subscriptions.Add(1)
		return nil
	}
	server.OnUnsubscribe = func(key string) {
		subscriptions.Sub(1)
	}
	server.Router.Handle("/metrics", promhttp.Handler())
}