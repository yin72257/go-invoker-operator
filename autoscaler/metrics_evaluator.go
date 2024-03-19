package autoscaler

type EvaluatedMetrics struct {
	PodMetrics    map[string]map[string]float32
	GlobalMetrics map[string]float32
}

func (metrics *EvaluatedMetrics) evaluate() {

}
