package controller

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yin72257/go-invoker-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func getDesiredClusterState(
	observed *ObservedInvokerDeploymentState,
	now time.Time, scheme *runtime.Scheme) DesiredInvokerDeploymentState {
	invokerDeployment := observed.cr

	if invokerDeployment == nil {
		return DesiredInvokerDeploymentState{}
	}
	return DesiredInvokerDeploymentState{
		ConfigMaps: getDesiredConfigMaps(invokerDeployment, scheme),
		Secret:     getDesiredSecret(invokerDeployment, scheme),
		// EntryService:     getDesiredEntryService(invokerDeployment, scheme),
		// Ingress:          getDesiredIngress(invokerDeployment, scheme),
		StatefulEntities: getDesiredStatefulEntities(invokerDeployment, scheme),
	}
}

func getDesiredConfigMaps(
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) []*corev1.ConfigMap {
	labels := labels(instance)
	labels["type"] = "statefulEntity"
	cmList := []*corev1.ConfigMap{}
	namespace := instance.ObjectMeta.Namespace

	for _, statefulEntity := range instance.Spec.StatefulEntities {
		configMapName := getConfigMapName(*statefulEntity.Name)
		data := map[string]string{
			"main": fmt.Sprintf(`
kafka.broker=%s
input.topic=%s
output.topic=%s
`, *statefulEntity.IOAddress, *statefulEntity.InputTopic, *statefulEntity.OutputTopic),
		}
		for _, podConfig := range statefulEntity.Pods {
			key := fmt.Sprintf(`%s.partitions`, *podConfig.Name)
			value := strings.Join(podConfig.Partitions, ",")
			data[key] = value
		}

		var configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      configMapName,
				Labels:    labels,
			},
			Data: data,
		}
		controllerutil.SetControllerReference(instance, configMap, scheme)
		cmList = append(cmList, configMap)
	}
	return cmList
}

func getDesiredDeployment(
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := labels(instance)
	name := instance.ObjectMeta.Name
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
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
							Name: "executor-pod",
							Env: []corev1.EnvVar{
								{
									Name: "KAFKA_BROKER",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: getConfigMapName(name),
											},
											Key: "broker.address",
										},
									},
								},
							},
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
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
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) *corev1.Secret {
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

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName()[0].Name,
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJson,
		},
	}
	controllerutil.SetControllerReference(instance, secret, scheme)
	return secret
}

func getDesiredEntryService(
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) *corev1.Service {

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
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) *networkingv1.Ingress {

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

func getDesiredStatefulSet(
	instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) *appsv1.StatefulSet {
	labels := labels(instance)
	name := instance.ObjectMeta.Name
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: getStatefulSetName(instance.Name),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "executor-pod",
							Env: []corev1.EnvVar{
								{
									Name: "KAFKA_BROKER",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: getConfigMapName(name),
											},
											Key: "broker.address",
										},
									},
								},
							},
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
						},
					},
					ImagePullSecrets: getSecretName(),
				},
			},
		},
	}

	controllerutil.SetControllerReference(instance, statefulSet, scheme)
	return statefulSet
}

func getDesiredStatefulEntities(instance *v1alpha1.InvokerDeployment, scheme *runtime.Scheme) []*corev1.Pod {

	podList := []*corev1.Pod{}

	for _, statefulEntity := range instance.Spec.StatefulEntities {
		labels := labels(instance)
		labels["type"] = "statefulEntity"
		labels["statefulEntityName"] = *statefulEntity.Name
		for _, podConfig := range statefulEntity.Pods {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *podConfig.Name,
					Namespace: instance.ObjectMeta.Namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "executor-pod",
							Image:           *statefulEntity.ExecutorImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/config",
								},
								{
									Name:      "uds-volume",
									MountPath: "/tmp",
								},
							},
							Resources: podConfig.Resources,
						},
						{
							Name:            "io-sidecar",
							Image:           *statefulEntity.IOImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Resources: podConfig.Resources,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/config",
								},
								{
									Name:      "uds-volume",
									MountPath: "/tmp",
								},
							},
						},
						{
							Name:            "state-sidecar",
							Image:           *statefulEntity.StateImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "uds-volume",
									MountPath: "/tmp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: getConfigMapName(*statefulEntity.Name),
									},
								},
							},
						},
						{
							Name: "uds-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			}

			controllerutil.SetControllerReference(instance, pod, scheme)
			podList = append(podList, pod)
		}

	}
	return podList
}
