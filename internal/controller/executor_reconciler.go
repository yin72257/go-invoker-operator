package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-logr/logr"
	"github.com/yin72257/go-executor-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ExecutorResourceReconciler struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	observed  ObservedExecutorState
	desired   DesiredExecutorState
	updater   ExecutorStatusUpdater
}

var requeueResult = ctrl.Result{RequeueAfter: 10 * time.Second, Requeue: true}

func (reconciler *ExecutorResourceReconciler) reconcile() (ctrl.Result, error) {
	var err error

	// Child resources of the cluster CR will be automatically reclaimed by K8S.
	if reconciler.observed.cr == nil {
		reconciler.log.Info("The cluster has been deleted, no action to take")
		return ctrl.Result{}, nil
	}
	// if getUpdateState(reconciler.observed) == UpdateStateInProgress {
	// 	reconciler.log.Info("The cluster update is in progress")
	// }
	err = reconciler.reconcileSecret()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileConfigMap()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileExecutorDeployment()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileEntryService()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileIngress()
	if err != nil {
		return ctrl.Result{}, err
	}

	var cluster = v1alpha1.Executor{}
	reconciler.observed.cr.DeepCopyInto(&cluster)
	cluster.Status.CurrentRevision = reconciler.observed.cr.Status.NextRevision
	err = reconciler.k8sClient.Status().Update(reconciler.context, &cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

func (reconciler *ExecutorResourceReconciler) reconcileExecutorDeployment() error {
	return reconciler.reconcileDeployment(
		"Executor",
		reconciler.desired.ExecutorDeployment,
		reconciler.observed.executorDeployment)
}

func (reconciler *ExecutorResourceReconciler) reconcileDeployment(
	component string,
	desiredDeployment *appsv1.Deployment,
	observedDeployment *appsv1.Deployment) error {
	var log = reconciler.log.WithValues("component", component)

	if desiredDeployment != nil && observedDeployment == nil {
		return reconciler.createDeployment(desiredDeployment, component)
	}

	if desiredDeployment != nil && observedDeployment != nil {
		if reconciler.observed.cr.Status.CurrentRevision != reconciler.observed.cr.Status.NextRevision {
			err := reconciler.updateComponent(desiredDeployment, "Deployment")
			if err != nil {
				return err
			}
			return nil
		}
		log.Info("Deployment already exists, no action")
		return nil
	}

	if desiredDeployment == nil && observedDeployment != nil {
		return reconciler.deleteDeployment(observedDeployment, component)
	}

	return nil
}

func (reconciler *ExecutorResourceReconciler) createDeployment(
	deployment *appsv1.Deployment, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating Deployment", "Deployment", *deployment)
	var err = k8sClient.Create(context, deployment)
	if err != nil {
		log.Error(err, "Failed to create Deployment")
	} else {
		log.Info("Deployment created")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) deleteOldComponent(desired runtime.Object, observed runtime.Object, component string) error {
	var log = reconciler.log.WithValues("component", component)
	var err error
	var context = reconciler.context
	var k8sClient = reconciler.k8sClient
	log.Info("Deleting component for update", "component", desired)

	desiredObj, ok := desired.(client.Object)
	if !ok {
		log.Error(err, "The desired object does not implement client.Object")
		return fmt.Errorf("the desired object does not implement client.Object")
	}
	err = k8sClient.Delete(context, desiredObj)
	if err != nil {
		log.Error(err, "Failed to delete component for update")
		return err
	}
	log.Info("Component deleted for update successfully")
	return nil
}

func (reconciler *ExecutorResourceReconciler) updateComponent(desired runtime.Object, component string) error {
	log := reconciler.log.WithValues("component", component)
	context := reconciler.context
	k8sClient := reconciler.k8sClient
	var err error

	log.Info("Update component", "component", desired)
	desiredObj, ok := desired.(client.Object)
	if !ok {
		log.Error(err, "The desired object does not implement client.Object")
		return fmt.Errorf("the desired object does not implement client.Object")
	}
	err = k8sClient.Update(context, desiredObj)
	if err != nil {
		log.Error(err, "Failed to update component for update")
		return err
	}
	log.Info("Component update successfully")
	return nil
}

func (reconciler *ExecutorResourceReconciler) deleteDeployment(
	deployment *appsv1.Deployment, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting Deployment", "Deployment", deployment)
	var err = k8sClient.Delete(context, deployment)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete Deployment")
	} else {
		log.Info("Deployment deleted")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) reconcileEntryService() error {
	var desiredEntryService = reconciler.desired.EntryService
	var observedEntryService = reconciler.observed.entryService

	if desiredEntryService != nil && observedEntryService == nil {
		return reconciler.createService(desiredEntryService, "Entry")
	}

	if desiredEntryService != nil && observedEntryService != nil {
		if reconciler.observed.cr.Status.CurrentRevision != reconciler.observed.cr.Status.NextRevision {
			err := reconciler.updateComponent(desiredEntryService, "Entry Serivce")
			if err != nil {
				return err
			}
			return nil
		}
		reconciler.log.Info("Entry service already exists, no action")
		return nil
	}

	if desiredEntryService == nil && observedEntryService != nil {
		return reconciler.deleteService(observedEntryService, "Entry")
	}

	return nil
}

func (reconciler *ExecutorResourceReconciler) createService(
	service *corev1.Service, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating service", "resource", *service)
	var err = k8sClient.Create(context, service)
	if err != nil {
		log.Info("Failed to create service", "error", err)
	} else {
		log.Info("Service created")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) deleteService(
	service *corev1.Service, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting service", "service", service)
	var err = k8sClient.Delete(context, service)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete service")
	} else {
		log.Info("service deleted")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) reconcileConfigMap() error {
	var desiredConfigMap = reconciler.desired.ConfigMap
	var observedConfigMap = reconciler.observed.configMap

	if desiredConfigMap != nil && observedConfigMap == nil {
		return reconciler.createConfigMap(desiredConfigMap, "ConfigMap")
	}

	if desiredConfigMap != nil && observedConfigMap != nil {
		if reconciler.observed.cr.Status.CurrentRevision != reconciler.observed.cr.Status.NextRevision {
			err := reconciler.updateComponent(desiredConfigMap, "ConfigMap")
			if err != nil {
				return err
			}
			return nil
		}
		reconciler.log.Info("ConfigMap already exists, no action")
		return nil
	}

	if desiredConfigMap == nil && observedConfigMap != nil {
		return reconciler.deleteConfigMap(observedConfigMap, "ConfigMap")
	}

	return nil
}

func (reconciler *ExecutorResourceReconciler) createConfigMap(
	cm *corev1.ConfigMap, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating configMap", "configMap", *cm)
	var err = k8sClient.Create(context, cm)
	if err != nil {
		log.Info("Failed to create configMap", "error", err)
	} else {
		log.Info("ConfigMap created")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) deleteConfigMap(
	cm *corev1.ConfigMap, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting configMap", "configMap", cm)
	var err = k8sClient.Delete(context, cm)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete configMap")
	} else {
		log.Info("ConfigMap deleted")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) reconcileSecret() error {
	var desiredSecret = reconciler.desired.Secret
	var observedSecret = reconciler.observed.secret

	if desiredSecret != nil && observedSecret == nil {
		return reconciler.createSecret(desiredSecret, "Secret")
	}

	if desiredSecret != nil && observedSecret != nil {
		reconciler.log.Info("Secret already exists, no action")
		return nil
	}

	if desiredSecret == nil && observedSecret != nil {
		return reconciler.deleteSecret(observedSecret, "Secret")
	}

	return nil
}

func (reconciler *ExecutorResourceReconciler) createSecret(
	secret *corev1.Secret, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating secret", "secret", *secret)
	var err = k8sClient.Create(context, secret)
	if err != nil {
		log.Info("Failed to create secret", "error", err)
	} else {
		log.Info("Secret created")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) deleteSecret(
	secret *corev1.Secret, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting secret", "secret", secret)
	var err = k8sClient.Delete(context, secret)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete secret")
	} else {
		log.Info("Secret deleted")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) reconcileIngress() error {
	var desiredIngress = reconciler.desired.Ingress
	var observedIngress = reconciler.observed.ingress

	if desiredIngress != nil && observedIngress == nil {
		return reconciler.createIngress(desiredIngress, "Ingress")
	}

	if desiredIngress != nil && observedIngress != nil {
		if reconciler.observed.cr.Status.CurrentRevision != reconciler.observed.cr.Status.NextRevision {
			err := reconciler.updateComponent(desiredIngress, "Ingress")
			if err != nil {
				return err
			}
			return nil
		}
		reconciler.log.Info("Ingress already exists, no action")
		return nil
	}

	if desiredIngress == nil && observedIngress != nil {
		return reconciler.deleteIngress(observedIngress, "Entry")
	}

	return nil
}

func (reconciler *ExecutorResourceReconciler) createIngress(
	ingress *networkingv1.Ingress, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating ingress", "resource", *ingress)
	var err = k8sClient.Create(context, ingress)
	if err != nil {
		log.Info("Failed to create ingress", "error", err)
	} else {
		log.Info("Ingress created")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) deleteIngress(
	ingress *networkingv1.Ingress, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting ingress", "ingress", ingress)
	var err = k8sClient.Delete(context, ingress)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete ingress")
	} else {
		log.Info("Ingress deleted")
	}
	return err
}

func (reconciler *ExecutorResourceReconciler) cleanupOldRevisions() error {
	var revisions appsv1.ControllerRevisionList
	selector := client.MatchingLabels{"invoker.io/executor": reconciler.observed.cr.Name}
	if err := reconciler.k8sClient.List(reconciler.context, &revisions, client.InNamespace(reconciler.observed.cr.Namespace), selector); err != nil {
		return err
	}

	if len(revisions.Items) <= int(*reconciler.observed.cr.Spec.HistoryLimit) {
		return nil // Nothing to do
	}

	sort.Slice(revisions.Items, func(i, j int) bool {
		return revisions.Items[i].Revision < revisions.Items[j].Revision
	})

	excess := len(revisions.Items) - int(*reconciler.observed.cr.Spec.HistoryLimit)

	for _, oldRevision := range revisions.Items[:excess] {
		if err := reconciler.k8sClient.Delete(reconciler.context, &oldRevision); err != nil {
			return err
		}
	}

	return nil
}
