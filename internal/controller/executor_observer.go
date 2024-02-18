package controller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	invokerv1alpha1 "github.com/yin72257/go-executor-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStateObserver gets the observed state of the cluster.
type ExecutorStateObserver struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
}

// ObservedClusterState holds observed state of a cluster.
type ObservedExecutorState struct {
	cr                 *invokerv1alpha1.Executor
	revisions          []*appsv1.ControllerRevision
	configMap          *corev1.ConfigMap
	executorDeployment *appsv1.Deployment
	secret             *corev1.Secret
	savepointErr       error
	observeTime        time.Time
	entryService       *corev1.Service
	ingress            *networkingv1.Ingress
}

type ExecutorStatus struct {
}

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *ExecutorStateObserver) observe(
	observed *ObservedExecutorState) error {
	var err error
	var log = observer.log

	// Cluster state.
	var observedCR = new(invokerv1alpha1.Executor)
	err = observer.observeSpec(observedCR)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get the executor resource")
			return err
		}
		log.Info("Observed executor", "executor", "nil")
		observedCR = nil
	} else {
		log.Info("Observed executor", "executor", *observedCR)
		observed.cr = observedCR
	}
	if observed.cr == nil {
		return nil
	}

	// Secret.
	var observedSecret = new(corev1.Secret)
	err = observer.observeSecret(observedSecret)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get secret")
			return err
		}
		log.Info("Observed secret", "state", "nil")
		observedSecret = nil
	} else {
		log.Info("Observed configMap", "state", *observedSecret)
		observed.secret = observedSecret
	}

	// ConfigMap.
	var observedConfigMap = new(corev1.ConfigMap)
	err = observer.observeConfigMap(observedConfigMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap")
			return err
		}
		log.Info("Observed configMap", "state", "nil")
		observedConfigMap = nil
	} else {
		log.Info("Observed configMap", "state", *observedConfigMap)
		observed.configMap = observedConfigMap
	}

	// Executor Deployment.
	var observedExecutorDeployment = new(appsv1.Deployment)
	err = observer.observeExecutorsDeployment(observedExecutorDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Executor Deployment")
			return err
		}
		log.Info("Observed Executor Deployment", "state", "nil")
		observedExecutorDeployment = nil
	} else {
		log.Info("Observed Executor Deployment", "state", *observedExecutorDeployment)
		observed.executorDeployment = observedExecutorDeployment
	}

	// Executor service.
	var observedEntryService = new(corev1.Service)
	err = observer.observeEntryService(observedEntryService)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Entry service")
			return err
		}
		log.Info("Observed Entry service", "state", "nil")
		observedEntryService = nil
	} else {
		log.Info("Observed Entry service", "state", *observedEntryService)
		observed.entryService = observedEntryService
	}

	// Executor ingress.
	var observedIngress = new(networkingv1.Ingress)
	err = observer.observeIngress(observedIngress)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Ingress")
			return err
		}
		log.Info("Observed Ingress", "state", "nil")
		observedIngress = nil
	} else {
		log.Info("Observed Ingress", "state", *observedIngress)
		observed.ingress = observedIngress
	}

	observed.observeTime = time.Now()

	specBytes, err := json.Marshal(observed.cr.Spec)
	if err != nil {
		return err
	}
	revisionName := generateRevisionName(observed.cr.Name, specBytes)
	observed.cr.Status.NextRevision = revisionName
	return nil
}

func (observer *ExecutorStateObserver) observeSpec(
	executor *invokerv1alpha1.Executor) error {
	return observer.k8sClient.Get(
		observer.context, observer.request.NamespacedName, executor)
}

func (observer *ExecutorStateObserver) observeConfigMap(
	observedConfigMap *corev1.ConfigMap) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getConfigMapName(clusterName),
		},
		observedConfigMap)
}

func (observer *ExecutorStateObserver) observeSecret(
	observedSecret *corev1.Secret) error {
	var clusterNamespace = observer.request.Namespace

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getSecretName()[0].Name,
		},
		observedSecret)
}

func (observer *ExecutorStateObserver) observeExecutorsDeployment(
	observedDeployment *appsv1.Deployment) error {
	var executorNamespace = observer.request.Namespace
	var executorName = observer.request.Name
	return observer.observeDeployment(
		executorNamespace, executorName, "Executor", observedDeployment)
}

func (observer *ExecutorStateObserver) observeDeployment(
	namespace string,
	name string,
	component string,
	observedDeployment *appsv1.Deployment) error {
	var log = observer.log.WithValues("component", component)
	var err = observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		observedDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Deployment")
		} else {
			log.Info("Deployment not found")
		}
	}
	return err
}

func (observer *ExecutorStateObserver) observeEntryService(
	observedService *corev1.Service) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getEntryServiceName(clusterName),
		},
		observedService)
}

func (observer *ExecutorStateObserver) observeIngress(
	observedIngress *networkingv1.Ingress) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getIngressName(clusterName),
		},
		observedIngress)
}
