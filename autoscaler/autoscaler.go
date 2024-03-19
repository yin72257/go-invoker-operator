package autoscaler

import (
	"context"

	invokerv1alpha1 "github.com/yin72257/go-executor-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Scaler struct {
	executorCR invokerv1alpha1.Executor
	k8sClient  client.Client
	context    context.Context
	log        logr.Logger
}

// Returns whether scaling decision was made
func (s *Scaler) scale() (bool, error) {
	s.log.Info("getting metrics for resource")
	//Get metrics
	s.log.Info("Retrieved metrics")
	var evaluatedMetrics = new(EvaluatedMetrics)
	evaluatedMetrics.evaluate()
	//Evaluate metrics
	s.log.Info("Evaluated metrics and compute new CR")
	//Evaluate new scaling
	changed, err := s.evaluateNewScaling(evaluatedMetrics)
	if err != nil {
		return false, err
	}

	return changed, nil
}

func (s *Scaler) evaluateNewScaling(metrics *EvaluatedMetrics) (bool, error) {
	cpu := metrics.GlobalMetrics["CPU"]
	newSpec := s.executorCR.DeepCopy()
	changed := false
	if int32(cpu) > 500 {
		changed = true
	}
	if changed {
		err := s.k8sClient.Update(s.context, newSpec)
		return changed, err
	}
	return changed, nil
}
