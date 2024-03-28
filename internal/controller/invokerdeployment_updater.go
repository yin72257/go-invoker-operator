package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/yin72257/go-invoker-operator/api/v1alpha1"
	controllermetrics "github.com/yin72257/go-invoker-operator/internal/controller/metrics"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InvokerDeploymentStatusUpdater struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	observed  ObservedInvokerDeploymentState
}

func (updater *InvokerDeploymentStatusUpdater) updateCondition(condition *metav1.Condition) error {
	var newStatus = v1alpha1.InvokerDeploymentStatus{}
	newStatus.DeepCopyInto(&updater.observed.cr.Status)
	meta.SetStatusCondition(&newStatus.Conditions, *condition)
	return updater.updateClusterStatus(newStatus)
}

func (updater *InvokerDeploymentStatusUpdater) updateStatusIfChanged() (
	bool, error) {
	if updater.observed.cr == nil {
		updater.log.Info("The resource has been deleted, no status to update")
		return false, nil
	}

	// Current status recorded in the cluster's status field.
	var oldStatus = v1alpha1.InvokerDeploymentStatus{}
	updater.observed.cr.Status.DeepCopyInto(&oldStatus)
	oldStatus.LastUpdateTime = ""

	// New status derived from the cluster's components.
	var newStatus = updater.deriveInvokerDeploymentStatus(
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

func (updater *InvokerDeploymentStatusUpdater) deriveInvokerDeploymentStatus(
	recorded *v1alpha1.InvokerDeploymentStatus,
	observed *ObservedInvokerDeploymentState) v1alpha1.InvokerDeploymentStatus {

	var status = v1alpha1.InvokerDeploymentStatus{}

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
	// InvokerDeploymentDeployment, entryService.
	var totalComponents = 3

	// ConfigMap.
	var observedConfigMap = observed.configMaps
	if len(observedConfigMap) != 0 {
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
	// InvokerDeployment StatefulEntities
	var observedStatefulEntities = observed.statefulEntities
	totalStatefulEntities := len(observedStatefulEntities)
	runningSE := 0
	for _, observedStatefulEntity := range observedStatefulEntities {
		if observedStatefulEntity != nil {
			allReady := true
			for _, observedPodInSE := range observedStatefulEntity {
				if getPodState(observedPodInSE) != v1alpha1.ComponentStateReady {
					allReady = false
				}
			}
			if allReady {
				runningSE++
			}
		}
	}
	if runningSE < totalStatefulEntities {
		status.Components.StatefulEntities = v1alpha1.ComponentStateNotReady
	} else {
		status.Components.StatefulEntities = v1alpha1.ComponentStateReady
	}
	if len(observedStatefulEntities) == 0 && recorded.Components.StatefulEntities != "" {
		status.Components.StatefulEntities = v1alpha1.ComponentStateDeleted
	}

	// Entry service.
	// var observedEntryService = observed.entryService
	// if observedEntryService != nil {
	// 	var state string
	// 	if observedEntryService.Spec.ClusterIP != "" {
	// 		state = v1alpha1.ComponentStateReady
	// 		runningComponents++
	// 	}
	// 	log.Info("Service Ready")
	// 	status.Components.EntryService = state
	// } else if recorded.Components.EntryService != "" {
	// 	status.Components.EntryService = v1alpha1.ComponentStateDeleted
	// }

	// Ingress.
	// var observedIngress = observed.ingress
	// if observedIngress != nil {
	// 	var state string
	// 	if observedIngress.ObjectMeta.Name != "" {
	// 		state = v1alpha1.ComponentStateReady
	// 		runningComponents++
	// 	}
	// 	log.Info("Ingress Ready")
	// 	status.Components.Ingress = state
	// } else if recorded.Components.Ingress != "" {
	// 	status.Components.Ingress = v1alpha1.ComponentStateDeleted
	// }
	isClusterStateUpdating := observed.cr.Status.CurrentRevision != observed.cr.Status.NextRevision
	// Derive the new cluster state.
	switch recorded.State {
	case "", v1alpha1.StateCreating:
		if runningComponents < totalComponents {
			status.State = v1alpha1.StateCreating
		} else {
			status.State = v1alpha1.StateRunning
		}
	case v1alpha1.StateRunning,
		v1alpha1.StateReconciling, v1alpha1.StateUpdating:
		if isClusterStateUpdating {
			log.Info("update metric : ApplicationUpgradeCounter")
			controllermetrics.ApplicationUpgradeCounter.Inc()
			status.State = v1alpha1.StateUpdating
		} else if runningComponents < totalComponents {
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

	status.CurrentRevision = observed.cr.Status.CurrentRevision
	status.NextRevision = observed.cr.Status.NextRevision

	return status
}

func (updater *InvokerDeploymentStatusUpdater) isStatusChanged(
	currentStatus v1alpha1.InvokerDeploymentStatus,
	newStatus v1alpha1.InvokerDeploymentStatus) bool {
	var changed = false
	if newStatus.State != currentStatus.State {
		changed = true
		updater.log.Info(
			"InvokerDeployment state changed",
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
	if newStatus.Components.StatefulEntities !=
		currentStatus.Components.StatefulEntities {
		updater.log.Info(
			"InvokerDeployment StatefulEntities status changed",
			"current", currentStatus.Components.StatefulEntities,
			"new",
			newStatus.Components.StatefulEntities)
		changed = true
	}
	// if newStatus.Components.EntryService !=
	// 	currentStatus.Components.EntryService {
	// 	updater.log.Info(
	// 		"Entry service status changed",
	// 		"current",
	// 		currentStatus.Components.EntryService,
	// 		"new", newStatus.Components.EntryService)
	// 	changed = true
	// }
	// if newStatus.Components.Ingress !=
	// 	currentStatus.Components.Ingress {
	// 	updater.log.Info(
	// 		"Ingress status changed",
	// 		"current",
	// 		currentStatus.Components.Ingress,
	// 		"new", newStatus.Components.Ingress)
	// 	changed = true
	// }
	return changed
}

func (updater *InvokerDeploymentStatusUpdater) updateClusterStatus(
	status v1alpha1.InvokerDeploymentStatus) error {
	log.Info("Updating status")
	var cluster = v1alpha1.InvokerDeployment{}
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

func getStatefulSetState(statefulSet *appsv1.StatefulSet) string {
	if statefulSet.Status.ReadyReplicas >= *statefulSet.Spec.Replicas {
		return v1alpha1.ComponentStateReady
	}
	return v1alpha1.ComponentStateNotReady
}

func getPodState(pod *corev1.Pod) string {
	if pod.Status.Phase == corev1.PodRunning {
		return v1alpha1.ComponentStateReady
	}
	return v1alpha1.ComponentStateNotReady
}

func timeToString(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}

func (updater *InvokerDeploymentStatusUpdater) syncRevisions(observed *ObservedInvokerDeploymentState) error {
	if observed.cr.Status.CurrentRevision == "" {
		return updater.createControllerRevision(observed, 1)
	}

	var lastRevision appsv1.ControllerRevision
	currentRevisionName := observed.cr.Status.CurrentRevision
	if err := updater.k8sClient.Get(updater.context, client.ObjectKey{Namespace: updater.observed.cr.Namespace, Name: currentRevisionName}, &lastRevision); err != nil {
		return err
	}
	var lastSpec *v1alpha1.InvokerDeploymentSpec
	if err := json.Unmarshal(lastRevision.Data.Raw, &lastSpec); err != nil {
		return err
	}

	if !reflect.DeepEqual(observed.cr.Spec, *lastSpec) {
		log.Info("creating revision:", "observed cr", observed.cr.Spec, "revision cr", lastSpec)
		return updater.createControllerRevision(observed, int(lastRevision.Revision)+1)
	}

	return nil
}

func (updater *InvokerDeploymentStatusUpdater) createControllerRevision(observed *ObservedInvokerDeploymentState, revisionNum int) error {
	specBytes, err := json.Marshal(observed.cr.Spec)
	if err != nil {
		return err
	}
	revisionName := generateRevisionName(observed.cr.Name, specBytes)
	observed.cr.Status.NextRevision = revisionName

	revision := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      revisionName,
			Namespace: observed.cr.Namespace,
			Labels: map[string]string{
				"invoker.io/invokerdeployment": observed.cr.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(observed.cr, observed.cr.GroupVersionKind()),
			},
		},
		Data: runtime.RawExtension{
			Raw: specBytes,
		},
		Revision: int64(revisionNum),
	}

	var currentRevision appsv1.ControllerRevision
	if err := updater.k8sClient.Get(updater.context, client.ObjectKey{Namespace: updater.observed.cr.Namespace, Name: revisionName}, &currentRevision); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := updater.k8sClient.Create(updater.context, revision); err != nil {
			updater.log.Error(err, "Failed to create revision", "currentRevision", observed.cr.Status.CurrentRevision)
			return err
		}
	} else {
		if err := updater.k8sClient.Update(updater.context, revision); err != nil {
			updater.log.Error(err, "Failed to update revision", "currentRevision", observed.cr.Status.CurrentRevision)
			return err
		}
	}

	var cluster = v1alpha1.InvokerDeployment{}
	updater.observed.cr.DeepCopyInto(&cluster)
	cluster.Status.NextRevision = revisionName
	return nil
}
