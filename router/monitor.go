package router

import (
	"github.com/benitogf/katamari"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
