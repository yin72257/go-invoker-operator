package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	invokerv1alpha1 "github.com/yin72257/go-invoker-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func labels(instance *invokerv1alpha1.InvokerDeployment) map[string]string {
	return map[string]string{
		"app": "invoker",
	}
}

func getConfigMapName(name string) string {
	return name + "-configmap"
}

func getExecutorDeploymentName(name string) string {
	return name + "-executor"
}

func getStatefulSetName(name string) string {
	return name + "-statefulset"
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
