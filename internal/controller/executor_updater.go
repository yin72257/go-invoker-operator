package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/yin72257/go-executor-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ExecutorStatusUpdater struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	observed  ObservedExecutorState
}

func (updater *ExecutorStatusUpdater) updateCondition(condition *metav1.Condition) error {
	var newStatus = v1alpha1.ExecutorStatus{}
	newStatus.DeepCopyInto(&updater.observed.cr.Status)
	meta.SetStatusCondition(&newStatus.Conditions, *condition)
	return updater.updateClusterStatus(newStatus)
}

func (updater *ExecutorStatusUpdater) updateStatusIfChanged() (
	bool, error) {
	if updater.observed.cr == nil {
		updater.log.Info("The resource has been deleted, no status to update")
		return false, nil
	}

	// Current status recorded in the cluster's status field.
	var oldStatus = v1alpha1.ExecutorStatus{}
	updater.observed.cr.Status.DeepCopyInto(&oldStatus)
	oldStatus.LastUpdateTime = ""

	// New status derived from the cluster's components.
	var newStatus = updater.deriveExecutorStatus(
		&updater.observed.cr.Status, &updater.observed)

	// Compare
	var changed = updater.isStatusChanged(oldStatus, newStatus)

	// Update
	if changed {
		updater.log.Info(
			"Status changed",
			"old",
			oldStatus,
			"new", newStatus)
		newStatus.LastUpdateTime = timeToString(time.Now())
		return true, updater.updateClusterStatus(newStatus)
	}

	updater.log.Info("No status change", "state", oldStatus.State)
	return false, nil
}

func (updater *ExecutorStatusUpdater) deriveExecutorStatus(
	recorded *v1alpha1.ExecutorStatus,
	observed *ObservedExecutorState) v1alpha1.ExecutorStatus {

	var status = v1alpha1.ExecutorStatus{}

	if recorded.Conditions == nil || len(recorded.Conditions) == 0 {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    typeAvailable,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation"})
	} else {
		status.Conditions = recorded.Conditions
	}
	var runningComponents = 0
	// executorDeployment, entryService.
	var totalComponents = 3

	// ConfigMap.
	var observedConfigMap = observed.configMap
	if observedConfigMap != nil {
		status.Components.ConfigMap = v1alpha1.ComponentStateReady
	} else if recorded.Components.ConfigMap != "" {
		status.Components.ConfigMap = v1alpha1.ComponentStateDeleted
	}

	// Secret.
	var observedSecret = observed.secret
	if observedSecret != nil {
		status.Components.Secret = v1alpha1.ComponentStateReady
	} else if recorded.Components.Secret != "" {
		status.Components.Secret = v1alpha1.ComponentStateDeleted
	}

	// Executor Deployment
	var observedDeployment = observed.executorDeployment
	if observedDeployment != nil {
		status.Components.ExecutorDeployment = getDeploymentState(observedDeployment)
		if status.Components.ExecutorDeployment == v1alpha1.ComponentStateReady {
			runningComponents++
			log.Info("Deployment Ready")
		}
	} else if recorded.Components.ExecutorDeployment != "" {
		status.Components.ExecutorDeployment = v1alpha1.ComponentStateDeleted
	}

	// Entry service.
	var observedEntryService = observed.entryService
	if observedEntryService != nil {
		var state string
		if observedEntryService.Spec.ClusterIP != "" {
			state = v1alpha1.ComponentStateReady
			runningComponents++
		}
		log.Info("Service Ready")
		status.Components.EntryService = state
	} else if recorded.Components.EntryService != "" {
		status.Components.EntryService = v1alpha1.ComponentStateDeleted
	}

	// Ingress.
	var observedIngress = observed.ingress
	if observedIngress != nil {
		var state string
		if observedIngress.ObjectMeta.Name != "" {
			state = v1alpha1.ComponentStateReady
			runningComponents++
		}
		log.Info("Ingress Ready")
		status.Components.Ingress = state
	} else if recorded.Components.Ingress != "" {
		status.Components.Ingress = v1alpha1.ComponentStateDeleted
	}
	// Derive the new cluster state.
	switch recorded.State {
	case "", v1alpha1.StateCreating:
		if runningComponents < totalComponents {
			status.State = v1alpha1.StateCreating
		} else {
			status.State = v1alpha1.StateRunning
		}
	case v1alpha1.StateRunning,
		v1alpha1.StateReconciling:
		if runningComponents < totalComponents {
			status.State = v1alpha1.StateReconciling
		} else {
			status.State = v1alpha1.StateRunning
		}
	case v1alpha1.StateStopping,
		v1alpha1.StateStopped:
		//TODO implementation
	default:
		panic(fmt.Sprintf("Unknown cluster state: %v", recorded.State))
	}

	return status
}

func (updater *ExecutorStatusUpdater) isStatusChanged(
	currentStatus v1alpha1.ExecutorStatus,
	newStatus v1alpha1.ExecutorStatus) bool {
	var changed = false
	if newStatus.State != currentStatus.State {
		changed = true
		updater.log.Info(
			"Executor state changed",
			"current",
			currentStatus.State,
			"new",
			newStatus.State)
	}
	if len(currentStatus.Conditions) != len(newStatus.Conditions) {
		changed = true
		updater.log.Info(
			"Conditions changed",
			"current",
			currentStatus.Conditions,
			"new",
			newStatus.Conditions)
	}
	if newStatus.Components.ConfigMap !=
		currentStatus.Components.ConfigMap {
		updater.log.Info(
			"ConfigMap status changed",
			"current",
			currentStatus.Components.ConfigMap,
			"new",
			newStatus.Components.ConfigMap)
		changed = true
	}
	if newStatus.Components.ExecutorDeployment !=
		currentStatus.Components.ExecutorDeployment {
		updater.log.Info(
			"Executor Deployment status changed",
			"current", currentStatus.Components.ExecutorDeployment,
			"new",
			newStatus.Components.ExecutorDeployment)
		changed = true
	}
	if newStatus.Components.EntryService !=
		currentStatus.Components.EntryService {
		updater.log.Info(
			"Entry service status changed",
			"current",
			currentStatus.Components.EntryService,
			"new", newStatus.Components.EntryService)
		changed = true
	}
	if newStatus.Components.Ingress !=
		currentStatus.Components.Ingress {
		updater.log.Info(
			"Ingress status changed",
			"current",
			currentStatus.Components.Ingress,
			"new", newStatus.Components.Ingress)
		changed = true
	}
	return changed
}

func (updater *ExecutorStatusUpdater) updateClusterStatus(
	status v1alpha1.ExecutorStatus) error {
	log.Info("Updating status")
	var cluster = v1alpha1.Executor{}
	updater.observed.cr.DeepCopyInto(&cluster)
	cluster.Status = status
	err := updater.k8sClient.Status().Update(updater.context, &cluster)
	return err
}

func getDeploymentState(deployment *appsv1.Deployment) string {
	if deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas {
		return v1alpha1.ComponentStateReady
	}
	return v1alpha1.ComponentStateNotReady
}

func timeToString(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}
