package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	synCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_conns_attempts",
		Help: "Number of attempted connections",
	})
)

func init() {
	prometheus.MustRegister(synCounter)
}

func prom() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}
