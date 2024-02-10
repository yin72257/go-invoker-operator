package controller

import (
	"fmt"
	"os"
	"time"

	"github.com/yin72257/go-executor-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func getDesiredClusterState(
	observed *ObservedExecutorState,
	now time.Time, scheme *runtime.Scheme) DesiredExecutorState {
	executor := observed.cr

	if executor == nil {
		return DesiredExecutorState{}
	}
	return DesiredExecutorState{
		ConfigMap:          getDesiredConfigMap(executor, scheme),
		ExecutorDeployment: getDesiredDeployment(executor, scheme),
		Secret:             getDesiredSecret(executor, scheme),
		EntryService:       getDesiredService(executor, scheme),
		Ingress:            getDesiredIngress(executor, scheme),
	}
}

func getDesiredConfigMap(
	instance *v1alpha1.Executor, scheme *runtime.Scheme) *corev1.ConfigMap {

	namespace := instance.ObjectMeta.Namespace
	name := instance.ObjectMeta.Name
	brokerPort := instance.Spec.Config.BrokerPort
	brokerIp := instance.Spec.Config.BrokerIp
	configMapName := getConfigMapName(name)
	labels := labels(instance)
	var configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
			Labels:    labels,
		},
		Data: map[string]string{
			"brokerPort": *brokerPort,
			"brokerIp":   *brokerIp,
		},
	}
	controllerutil.SetControllerReference(instance, configMap, scheme)
	return configMap
}

func getDesiredDeployment(
	instance *v1alpha1.Executor, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := labels(instance)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "executor-pod",
							Image: *instance.Spec.Image,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_BROKER",
									Value: address(instance),
								},
							},
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
							Resources:       instance.Spec.Resource,
						},
					},
					ImagePullSecrets: getSecretName(),
				},
			},
		},
	}

	controllerutil.SetControllerReference(instance, dep, scheme)
	return dep
}

func getDesiredSecret(
	instance *v1alpha1.Executor, scheme *runtime.Scheme) *corev1.Secret {
	dockerPassword := os.Getenv("PASSWORD")
	dockerUsername := os.Getenv("USERNAME")
	dockerEmail := os.Getenv("EMAIL")
	dockerConfigJson := []byte(fmt.Sprintf(`{
			"auths": {
				"%s": {
					"username": "%s",
					"password": "%s",
					"email": "%s"
				}
			}
		}`, "https://index.docker.io/v1/", dockerUsername, dockerPassword, dockerEmail))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName()[0].Name,
			Namespace: instance.Name,
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			v1.DockerConfigJsonKey: dockerConfigJson,
		},
	}
	controllerutil.SetControllerReference(instance, secret, scheme)
	return secret
}

func getDesiredService(
	instance *v1alpha1.Executor, scheme *runtime.Scheme) *corev1.Service {

	namespace := instance.ObjectMeta.Namespace
	name := instance.ObjectMeta.Name
	serviceName := getEntryServiceName(name)
	labels := labels(instance)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080), // target port to port on application
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	controllerutil.SetControllerReference(instance, service, scheme)
	return service
}

func getDesiredIngress(
	instance *v1alpha1.Executor, scheme *runtime.Scheme) *networkingv1.Ingress {

	namespace := instance.ObjectMeta.Namespace
	name := instance.ObjectMeta.Name
	ingressName := getIngressName(name)
	serviceName := getEntryServiceName(name)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "invoker.io",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	controllerutil.SetControllerReference(instance, ingress, scheme)
	return ingress
}
