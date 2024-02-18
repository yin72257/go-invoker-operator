/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	invokerv1alpha1 "github.com/yin72257/go-executor-operator/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
)

var log = ctrllog.Log.WithName("controller_executor")

const executorFinalizer = "executor.invoker.io/finalizer"

const (
	typeAvailable = "Available"
	typeDegraded  = "Degraded"
)

// ExecutorReconciler reconciles a Executor object
type ExecutorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type DesiredExecutorState struct {
	ExecutorDeployment *appsv1.Deployment
	EntryService       *v1.Service
	ConfigMap          *v1.ConfigMap
	Secret             *v1.Secret
	Ingress            *networkingv1.Ingress
}

type ExecutorHandler struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
	recorder  record.EventRecorder
	observed  ObservedExecutorState
	desired   DesiredExecutorState
}

//+kubebuilder:rbac:groups=executor.invoker.io,resources=executors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=executor.invoker.io,resources=executors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=executor.invoker.io,resources=executors/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;
//+kubebuilder:rbac:groups=apps,resources=controllerrevisions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=controllerrevisions/status,verbs=get;
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Executor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ExecutorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	log.Info("Reconciling")

	handler := ExecutorHandler{
		k8sClient: r.Client,
		request:   req,
		context:   ctx,
		log:       log,
		observed:  ObservedExecutorState{},
		recorder:  r.Recorder,
	}
	return handler.reconcile(req, r.Scheme)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&invokerv1alpha1.Executor{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (handler *ExecutorHandler) reconcile(
	request ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	var k8sClient = handler.k8sClient
	var log = handler.log
	var context = handler.context
	var observed = &handler.observed
	var desired = &handler.desired
	var statusChanged bool
	var err error

	log.Info("============================================================")
	log.Info("---------- 1. Observe the current state ----------")

	var observer = ExecutorStateObserver{
		k8sClient: k8sClient,
		request:   request,
		context:   context,
		log:       log,
	}
	err = observer.observe(observed)
	if err != nil {
		log.Error(err, "Failed to observe the current state")
		return ctrl.Result{}, err
	}
	if observed.cr == nil {
		log.Info("The resource has been deleted")
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(observed.cr, executorFinalizer) {
		log.Info("Adding finalizer for executor")
		if ok := controllerutil.AddFinalizer(observed.cr, executorFinalizer); !ok {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, nil
		}
		if err := k8sClient.Update(context, observed.cr); err != nil {
			log.Error(err, "Failed to add finalizer to CR")
			return ctrl.Result{}, err
		}
	}

	log.Info("---------- 2. Update executor status ----------")

	var updater = ExecutorStatusUpdater{
		k8sClient: k8sClient,
		context:   context,
		log:       log,
		observed:  handler.observed,
	}
	if observed.cr.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(observed.cr, executorFinalizer) {
			if err = updater.updateCondition(&metav1.Condition{
				Type:    typeDegraded,
				Status:  metav1.ConditionUnknown,
				Reason:  "Finalizing",
				Message: "Performing finalizing operations"}); err != nil {
				log.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}

			handler.doFinalizeOperation()

			if err = observer.observe(observed); err != nil {
				log.Error(err, "Failed to re-fetch Executor")
				return ctrl.Result{}, err
			}

			if err = updater.updateCondition(&metav1.Condition{
				Type:    typeDegraded,
				Status:  metav1.ConditionUnknown,
				Reason:  "Finalizing",
				Message: "Finished performing finalizing operations"}); err != nil {
				log.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}

			if err = observer.observe(observed); err != nil {
				log.Error(err, "Failed to re-fetch Executor")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer after successful operation")
			if ok := controllerutil.RemoveFinalizer(observed.cr, executorFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := k8sClient.Update(context, observed.cr); err != nil {
				log.Error(err, "Failed to remove finalizer on update")
				return ctrl.Result{}, err
			}
		}
		log.Info("CR Marked as to be deleted")
		return ctrl.Result{}, nil
	}

	statusChanged, err = updater.updateStatusIfChanged()
	if err != nil {
		log.Error(err, "Failed to update executor status")
		return ctrl.Result{}, err
	}
	if statusChanged {
		log.Info(
			"Wait status to be stable before taking further actions.",
			"requeueAfter",
			5)
		return ctrl.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		}, nil
	}

	log.Info("---------- 3. Compute the desired state ----------")

	*desired = getDesiredClusterState(observed, time.Now(), scheme)

	if desired.ConfigMap != nil {
		log.Info("Desired state", "ConfigMap", *desired.ConfigMap)
	} else {
		log.Info("Desired state", "ConfigMap", "nil")
	}
	if desired.ExecutorDeployment != nil {
		log.Info("Desired state", "Executor Deployment", *desired.ExecutorDeployment)
	} else {
		log.Info("Desired state", "Executor Deployment", "nil")
	}
	if desired.Secret != nil {
		log.Info("Desired state", "Secret", *desired.Secret)
	} else {
		log.Info("Desired state", "Secret", "nil")
	}
	if desired.EntryService != nil {
		log.Info("Desired state", "Entry Service", *desired.EntryService)
	} else {
		log.Info("Desired state", "Entry Service", "nil")
	}
	if desired.Ingress != nil {
		log.Info("Desired state", "Ingress", *desired.Ingress)
	} else {
		log.Info("Desired state", "Ingress", "nil")
	}

	log.Info("---------- 4. Take actions ----------")

	var reconciler = ExecutorResourceReconciler{
		k8sClient: k8sClient,
		context:   context,
		log:       log,
		observed:  handler.observed,
		desired:   handler.desired,
	}
	result, err := reconciler.reconcile()
	if err != nil {
		log.Error(err, "Failed to reconcile")
	}
	err = updater.syncRevisions(observed)
	if err != nil {
		log.Error(err, "Failed to sync revisions")
		return ctrl.Result{}, err
	}
	err = reconciler.cleanupOldRevisions()
	if err != nil {
		log.Error(err, "Failed to cleanup old revisions")
	}
	if result.RequeueAfter > 0 {
		log.Info("Requeue reconcile request", "after", result.RequeueAfter)
	}

	return result, err
}

func (handler *ExecutorHandler) doFinalizeOperation() {
	handler.recorder.Event(handler.observed.cr, "Warning", "Deleting", fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
		handler.request.Name,
		handler.request.Namespace))
}
