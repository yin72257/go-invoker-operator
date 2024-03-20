package controller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	invokerv1alpha1 "github.com/yin72257/go-invoker-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InvokerDeploymentStateObserver gets the observed state of the cluster.
type InvokerDeploymentStateObserver struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
}

// ObservedInvokerDeploymentState holds observed state of a cluster.
type ObservedInvokerDeploymentState struct {
	cr                           *invokerv1alpha1.InvokerDeployment
	revisions                    []*appsv1.ControllerRevision
	configMaps                   map[string]*corev1.ConfigMap
	invokerDeploymentStatefulSet *appsv1.StatefulSet
	secret                       *corev1.Secret
	observeTime                  time.Time
	entryService                 *corev1.Service
	statefulSetService           *corev1.Service
	ingress                      *networkingv1.Ingress
	statefulEntities             map[string]*appsv1.StatefulSet
}

type InvokerDeploymentStatus struct {
}

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *InvokerDeploymentStateObserver) observe(
	observed *ObservedInvokerDeploymentState) error {
	var err error
	var log = observer.log

	var observedCR = new(invokerv1alpha1.InvokerDeployment)
	err = observer.observeSpec(observedCR)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get the invokerDeployment resource")
			return err
		}
		log.Info("Observed invokerDeployment", "invokerDeployment", "nil")
		observedCR = nil
	} else {
		log.Info("Observed invokerDeployment", "invokerDeployment", *observedCR)
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
	var observedConfigMaps = new(corev1.ConfigMapList)
	err = observer.observeConfigMaps(observedConfigMaps)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap")
			return err
		}
		log.Info("Observed configMaps", "state", "nil")
		observedConfigMaps = nil
	} else {
		observed.configMaps = make(map[string]*corev1.ConfigMap)
		log.Info("Observed configMaps", "state", *observedConfigMaps)
		for i := range observedConfigMaps.Items {
			observed.configMaps[observedConfigMaps.Items[i].Name] = &observedConfigMaps.Items[i]
		}
	}

	// InvokerDeployment StatefulSet.
	var observedInvokerDeploymentStatefulSet = new(appsv1.StatefulSet)
	err = observer.observeInvokerDeploymentStatefulSet(observedInvokerDeploymentStatefulSet)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get InvokerDeployment StatefulSet")
			return err
		}
		log.Info("Observed InvokerDeployment StatefulSet", "state", "nil")
		observedInvokerDeploymentStatefulSet = nil
	} else {
		log.Info("Observed InvokerDeployment StatefulSet", "state", *observedInvokerDeploymentStatefulSet)
		observed.invokerDeploymentStatefulSet = observedInvokerDeploymentStatefulSet
	}

	// InvokerDeployment service.
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

	// InvokerDeployment stateful set service.
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

	// InvokerDeployment ingress.
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

	// StatefulEntities
	var observedStatefulEntities = new(appsv1.StatefulSetList)
	err = observer.observeStatefulEntities(observedStatefulEntities)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get list SE")
			return err
		}
		log.Info("Observed list SE", "state", "nil")
		observedStatefulEntities = nil
	} else {
		log.Info("Observed SE", "state", *observedStatefulEntities)
		observed.statefulEntities = make(map[string]*appsv1.StatefulSet)
		for i := range observedStatefulEntities.Items {
			observed.statefulEntities[observedStatefulEntities.Items[i].Name] = &observedStatefulEntities.Items[i]
		}
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

func (observer *InvokerDeploymentStateObserver) observeRevisions(revisions *appsv1.ControllerRevisionList) error {
	selector := client.MatchingLabels{"invoker.io/invokerDeployment": observer.request.Name}
	if err := observer.k8sClient.List(observer.context, revisions, client.InNamespace(observer.request.Namespace), selector); err != nil {
		return err
	}
	return nil
}

func (observer *InvokerDeploymentStateObserver) observeSpec(
	invokerDeployment *invokerv1alpha1.InvokerDeployment) error {
	return observer.k8sClient.Get(
		observer.context, observer.request.NamespacedName, invokerDeployment)
}

func (observer *InvokerDeploymentStateObserver) observeConfigMaps(
	observedConfigMaps *corev1.ConfigMapList) error {
	var namespace = observer.request.Namespace
	labelSelector := client.MatchingLabels{"type": "statefulEntity"}
	return observer.k8sClient.List(observer.context, observedConfigMaps, labelSelector, client.InNamespace(namespace))
}

func (observer *InvokerDeploymentStateObserver) observeSecret(
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

func (observer *InvokerDeploymentStateObserver) observeInvokerDeploymentStatefulSet(
	observedStatefulSet *appsv1.StatefulSet) error {
	var invokerDeploymentNamespace = observer.request.Namespace
	var invokerDeploymentName = observer.request.Name
	return observer.observeStatefulSet(
		invokerDeploymentNamespace, invokerDeploymentName, "InvokerDeployment", observedStatefulSet)
}

func (observer *InvokerDeploymentStateObserver) observeDeployment(
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

func (observer *InvokerDeploymentStateObserver) observeStatefulSet(
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

func (observer *InvokerDeploymentStateObserver) observeService(
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

func (observer *InvokerDeploymentStateObserver) observeIngress(
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

func (observer *InvokerDeploymentStateObserver) observeStatefulEntities(observedStatefulEntities *appsv1.StatefulSetList) error {
	var namespace = observer.request.Namespace
	labelSelector := client.MatchingLabels{"type": "statefulEntity"}
	if err := observer.k8sClient.List(observer.context, observedStatefulEntities, labelSelector, client.InNamespace(namespace)); err != nil {
		return err
	}
	return nil
}
