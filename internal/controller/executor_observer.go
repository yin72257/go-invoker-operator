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

// ExecutorStateObserver gets the observed state of the cluster.
type ExecutorStateObserver struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
}

// ObservedExecutorState holds observed state of a cluster.
type ObservedExecutorState struct {
	cr                  *invokerv1alpha1.Executor
	revisions           []*appsv1.ControllerRevision
	configMap           *corev1.ConfigMap
	executorStatefulSet *appsv1.StatefulSet
	secret              *corev1.Secret
	observeTime         time.Time
	entryService        *corev1.Service
	statefulSetService  *corev1.Service
	ingress             *networkingv1.Ingress
}

type ExecutorStatus struct {
}

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *ExecutorStateObserver) observe(
	observed *ObservedExecutorState) error {
	var err error
	var log = observer.log

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

	// Executor StatefulSet.
	var observedExecutorStatefulSet = new(appsv1.StatefulSet)
	err = observer.observeExecutorStatefulSet(observedExecutorStatefulSet)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Executor StatefulSet")
			return err
		}
		log.Info("Observed Executor StatefulSet", "state", "nil")
		observedExecutorStatefulSet = nil
	} else {
		log.Info("Observed Executor StatefulSet", "state", *observedExecutorStatefulSet)
		observed.executorStatefulSet = observedExecutorStatefulSet
	}

	// Executor service.
	var observedEntryService = new(corev1.Service)
	err = observer.observeService(observedEntryService, getEntryServiceName(observer.request.Name))
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

	// Executor stateful set service.
	var observedStatefulSetService = new(corev1.Service)
	err = observer.observeService(observedStatefulSetService, getStatefulSetName(observer.request.Name))
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get StatefulSet service")
			return err
		}
		log.Info("Observed StatefulSet service", "state", "nil")
		observedStatefulSetService = nil
	} else {
		log.Info("Observed StatefulSet service", "state", *observedStatefulSetService)
		observed.statefulSetService = observedStatefulSetService
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

	var observedRevisions = new(appsv1.ControllerRevisionList)
	err = observer.observeRevisions(observedRevisions)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Revisions")
			return err
		}
		log.Info("Observed Revisions", "state", "nil")
		observedRevisions = nil
	} else {
		log.Info("Observed Revisions", "state", *observedRevisions)
		for i := range observedRevisions.Items {
			observed.revisions = append(observed.revisions, &observedRevisions.Items[i])
		}
	}

	specBytes, err := json.Marshal(observed.cr.Spec)
	if err != nil {
		return err
	}
	revisionName := generateRevisionName(observed.cr.Name, specBytes)
	observed.cr.Status.NextRevision = revisionName
	return nil
}

func (observer *ExecutorStateObserver) observeRevisions(revisions *appsv1.ControllerRevisionList) error {
	selector := client.MatchingLabels{"invoker.io/executor": observer.request.Name}
	if err := observer.k8sClient.List(observer.context, revisions, client.InNamespace(observer.request.Namespace), selector); err != nil {
		return err
	}
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

func (observer *ExecutorStateObserver) observeExecutorStatefulSet(
	observedStatefulSet *appsv1.StatefulSet) error {
	var executorNamespace = observer.request.Namespace
	var executorName = observer.request.Name
	return observer.observeStatefulSet(
		executorNamespace, executorName, "Executor", observedStatefulSet)
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

func (observer *ExecutorStateObserver) observeStatefulSet(
	namespace string,
	name string,
	component string,
	observedStatefulSet *appsv1.StatefulSet) error {
	var log = observer.log.WithValues("component", component)
	var err = observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		observedStatefulSet)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get StatefulSet")
		} else {
			log.Info("StatefulSet not found")
		}
	}
	return err
}

func (observer *ExecutorStateObserver) observeService(
	observedService *corev1.Service, serviceName string) error {
	var namespace = observer.request.Namespace

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      serviceName,
		},
		observedService)
}

func (observer *ExecutorStateObserver) observeIngress(
	observedIngress *networkingv1.Ingress) error {
	var namespace = observer.request.Namespace
	var name = observer.request.Name

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      getIngressName(name),
		},
		observedIngress)
}
