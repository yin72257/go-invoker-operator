package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	invokerv1alpha1 "github.com/yin72257/go-executor-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ExecutorReconciler) updateCondition(instance *invokerv1alpha1.Executor, condition *metav1.Condition) error {
	meta.SetStatusCondition(&instance.Status.Conditions, *condition)
	if err := r.Status().Update(context.TODO(), instance); err != nil {
		log.Error(err, "Failed to update Memcached status")
		return err
	}
	return nil
}

func labels(instance *invokerv1alpha1.Executor) map[string]string {
	return map[string]string{
		"app": "executor",
	}
}

func address(instance *invokerv1alpha1.Executor) string {
	return *instance.Spec.Config.BrokerIp + ":" + *instance.Spec.Config.BrokerPort
}

func getConfigMapName(name string) string {
	return name + "-configmap"
}

func getExecutorDeploymentName(name string) string {
	return name + "-executor"
}

func getEntryServiceName(name string) string {
	return name + "-entry"
}

func getIngressName(name string) string {
	return name + "-ingress"
}

func getSecretName() []corev1.LocalObjectReference {
	return []corev1.LocalObjectReference{
		{
			Name: "my-docker-registry-secret",
		},
	}
}

func generateRevisionName(resourceName string, specBytes []byte) string {
	hash := sha256.Sum256(specBytes)
	hexHash := hex.EncodeToString(hash[:])
	shortHash := hexHash[:8]
	revisionName := fmt.Sprintf("%s-rev-%s", resourceName, shortHash)
	return revisionName
}
