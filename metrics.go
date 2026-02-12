package gojinn

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type gojinnMetrics struct {
	duration   *prometheus.HistogramVec
	active     *prometheus.GaugeVec
	queueDepth *prometheus.GaugeVec
	jobsTotal  *prometheus.CounterVec
}

func (r *Gojinn) setupMetrics(ctx caddy.Context) error {
	r.metrics = &gojinnMetrics{}
	registry := ctx.GetMetricsRegistry()

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gojinn_function_duration_seconds",
		Help:    "Time taken to execute the WASM function",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "status"})

	if err := registry.Register(duration); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.duration = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			return fmt.Errorf("failed to register duration metric: %v", err)
		}
	} else {
		r.metrics.duration = duration
	}

	active := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gojinn_active_sandboxes",
		Help: "Number of WASM sandboxes currently running",
	}, []string{"path"})

	if err := registry.Register(active); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.active = are.ExistingCollector.(*prometheus.GaugeVec)
		} else {
			return fmt.Errorf("failed to register active metric: %v", err)
		}
	} else {
		r.metrics.active = active
	}

	queueDepth := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gojinn_worker_queue_depth",
		Help: "Number of pending jobs in the NATS JetStream stream",
	}, []string{"stream"})

	if err := registry.Register(queueDepth); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.queueDepth = are.ExistingCollector.(*prometheus.GaugeVec)
		} else {
			return fmt.Errorf("failed to register queueDepth metric: %v", err)
		}
	} else {
		r.metrics.queueDepth = queueDepth
	}

	jobsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gojinn_worker_jobs_total",
		Help: "Total number of worker jobs processed by status",
	}, []string{"status"})

	if err := registry.Register(jobsTotal); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.jobsTotal = are.ExistingCollector.(*prometheus.CounterVec)
		} else {
			return fmt.Errorf("failed to register jobsTotal metric: %v", err)
		}
	} else {
		r.metrics.jobsTotal = jobsTotal
	}

	return nil
}
