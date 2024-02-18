package controller_metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	ApplicationUpgradeCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "executor_upgrade_counter",
			Help: "Number of successful upgrades processed",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(ApplicationUpgradeCounter)
}
