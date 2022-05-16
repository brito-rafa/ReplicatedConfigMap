/*
Copyright 2022 Rafael Brito.

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

package controllers

import (
	"context"

	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	replicationsv1alpha1 "rcm/api/v1alpha1"
	"time"
)

const (
	addRcmDataAnnotation       = "rcm.replications.example.io/hash-data"
	addRcmBinaryDataAnnotation = "rcm.replications.example.io/hash-binarydata"
	rcmLabel                   = "rcm-sync"
	finalizer                  = "replications.example.io/finalizers"
	defaultRequeue             = 10 * time.Second
)

// ReplicatedConfigMapReconciler reconciles a ReplicatedConfigMap object
type ReplicatedConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=replications.example.io,resources=replicatedconfigmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=replications.example.io,resources=replicatedconfigmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=replications.example.io,resources=replicatedconfigmaps/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ReplicatedConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ReplicatedConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	myLog := log.FromContext(ctx)

	msg := fmt.Sprintf("received reconcile request for rcm %q, namespace %q, namespacedname %q", req.Name, req.Namespace, req.NamespacedName)
	myLog.Info(msg)

	rcmObj := &replicationsv1alpha1.ReplicatedConfigMap{}

	// rcm is cluster scoped, but Client Get requires a namespace as parameter
	if err := r.Client.Get(ctx, req.NamespacedName, rcmObj); err != nil {
		if !k8serr.IsNotFound(err) {
			myLog.Error(err, "unable to fetch ReplicatedConfigMap")
		}
		rcmObj.Status.Phase = replicationsv1alpha1.PhaseFailed
		r.Status().Update(ctx, rcmObj)

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg = fmt.Sprintf("able to parse %q", rcmObj.GetName())
	myLog.Info(msg)

	if rcmObj.Status.Phase == "" {
		rcmObj.Status.Phase = replicationsv1alpha1.PhaseNew
		if err := r.Status().Update(ctx, rcmObj); err != nil {
			myLog.Error(err, "unable to update ReplicatedConfigMap status")
			return ctrl.Result{}, err
		}
	}

	var hmDataMd5 = "empty"
	var hmd5BinaryData = "empty"

	/*
		if rcmObj.Data != nil {
			hmd5DataSum := md5.Sum([]byte(fmt.Sprint(rcmObj.Data)))
			hmDataMd5 = fmt.Sprint(hmd5DataSum)
		}

		if rcmObj.BinaryData != nil {
			hmd5BinaryDataSum := md5.Sum([]byte(fmt.Sprint(rcmObj.BinaryData)))
			hmd5BinaryData = fmt.Sprint(hmd5BinaryDataSum)
		}
	*/

	ann := rcmObj.Annotations

	if ann == nil {
		ann = make(map[string]string)
	}

	if ann[addRcmDataAnnotation] != hmDataMd5 {
		ann[addRcmDataAnnotation] = hmDataMd5
		msg = fmt.Sprintf("Hash data on %q changed", rcmObj.GetName())
		myLog.Info(msg)
	}

	if ann[addRcmBinaryDataAnnotation] != hmd5BinaryData {
		ann[addRcmBinaryDataAnnotation] = hmd5BinaryData
		msg = fmt.Sprintf("Hash data binary on %q changed", rcmObj.GetName())
		myLog.Info(msg)
	}

	rcmObj.Annotations = ann

	// Handling finalizer
	if rcmObj.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(rcmObj.GetFinalizers(), finalizer) {
			myLog.Info("Adding finalizer", "name", finalizer)
			controllerutil.AddFinalizer(rcmObj, finalizer)
			if err := r.Update(ctx, rcmObj); err != nil {
				myLog.Error(err, "unable to add finalizer")
				rcmObj.Status.Phase = replicationsv1alpha1.PhaseFailed
				r.Status().Update(ctx, rcmObj)
				return ctrl.Result{}, err
			}
		}
	} else {
		// Deleting resources and CR
		rcmObj.Status.Phase = replicationsv1alpha1.PhaseDeletion
		r.Status().Update(ctx, rcmObj)
		if containsString(rcmObj.GetFinalizers(), finalizer) {
			msg = fmt.Sprintf("Finalizer deleting resources on %q", rcmObj.GetName())
			myLog.Info(msg)
			if err := r.doDeleteResources(ctx, rcmObj); err != nil {
				myLog.Error(err, "unable to delete child resources")
				rcmObj.Status.Phase = replicationsv1alpha1.PhaseFailed
				r.Status().Update(ctx, rcmObj)
				return ctrl.Result{}, err
			}
			msg = fmt.Sprintf("Finalizer complete on %q", rcmObj.GetName())
			myLog.Info(msg)
			controllerutil.RemoveFinalizer(rcmObj, finalizer)
			// for some reason, the finalizer is not getting removed
			// trying two updates
			r.Status().Update(ctx, rcmObj)
			controllerutil.RemoveFinalizer(rcmObj, finalizer)
			if err := r.Status().Update(ctx, rcmObj); err != nil {
				rcmObj.Status.Phase = replicationsv1alpha1.PhaseFailed
				return ctrl.Result{}, err
			}
		}
	}

	res := ctrl.Result{RequeueAfter: defaultRequeue}
	var err error

	// If Deletion is in process, do not attempt to reconcile
	if rcmObj.Status.Phase != replicationsv1alpha1.PhaseDeletion {
		res, err = r.doReconcileResources(ctx, rcmObj)
		if err != nil {
			myLog.Error(err, "could not finish reconciliation)")
			rcmObj.Status.Phase = replicationsv1alpha1.PhaseFailed
			r.Status().Update(ctx, rcmObj)
			return ctrl.Result{}, err
		}
	} else {
		// wait a bit for the requeue
		return ctrl.Result{RequeueAfter: defaultRequeue}, nil
	}

	r.Client.Status().Update(ctx, rcmObj)
	msg = fmt.Sprintf("Reconcile complete on %q", rcmObj.GetName())
	myLog.Info(msg)

	return res, nil
}

func (r *ReplicatedConfigMapReconciler) doReconcileResources(ctx context.Context, obj *replicationsv1alpha1.ReplicatedConfigMap) (ctrl.Result, error) {

	myLog := log.FromContext(ctx)

	msg := fmt.Sprintf("Looking for child resources of %q for reconciliation", obj.GetName())
	myLog.Info(msg)

	obj.Status.Phase = replicationsv1alpha1.PhaseInProgress
	if err := r.Status().Update(ctx, obj); err != nil {
		myLog.Error(err, "unable to update ReplicatedConfigMap status")
		return ctrl.Result{}, err
	}

	obj.Status.MatchingNamespaces = ""

	for _, namespace := range r.findNamespaces(ctx, obj) {

		cm := r.getCm(ctx, obj, namespace)

		if cm == nil {
			msg := fmt.Sprintf("Cannot find config map %q on namespace %q. Creating one", obj.GetName(), namespace)
			myLog.Info(msg)

			cm := &corev1.ConfigMap{}
			cm.Kind = "ConfigMap"
			cm.APIVersion = "v1"
			cm.Name = obj.GetName()
			cm.Namespace = namespace
			cm.Annotations = map[string]string{
				addRcmDataAnnotation:       obj.Annotations[addRcmDataAnnotation],
				addRcmBinaryDataAnnotation: obj.Annotations[addRcmBinaryDataAnnotation],
			}
			if obj.Data != nil {
				cm.Data = obj.Data
			}
			if obj.BinaryData != nil {
				cm.BinaryData = obj.BinaryData
			}
			cm.Labels = map[string]string{
				rcmLabel: "true",
			}

			err := r.Create(ctx, cm)

			if err != nil {
				obj.Status.Phase = replicationsv1alpha1.PhaseFailed
				myLog.Error(err, "unable to create configmap")
				r.Status().Update(ctx, obj)
				return ctrl.Result{}, err
			}

		} else {

			var hmd5BinaryData = "empty"
			var cmMd5 = "empty"

			/*
				if &cm.Data != nil {
					hmd5DataSum := md5.Sum([]byte(fmt.Sprint(&cm.Data)))
					cmMd5 = fmt.Sprint(hmd5DataSum)
				}
			*/

			/*
				if &cm.BinaryData != nil {
					hmd5BinaryDataSum := md5.Sum([]byte(fmt.Sprint(&cm.BinaryData)))
					hmd5BinaryData = fmt.Sprint(hmd5BinaryDataSum)
				}
			*/

			//if cmMd5 != obj.Annotations[addRcmDataAnnotation] {
			// || hmd5BinaryData != obj.Annotations[addRcmBinaryDataAnnotation]

			//msg := fmt.Sprintf("Config map %q on namespace %q has different hashes, updating it", obj.GetName(), namespace)
			//myLog.Info(msg)

			msg = fmt.Sprintf("data hash cm %q binary hash cm %q data hash obj, data hash obj %q binary data obj %q", cmMd5, hmd5BinaryData, obj.Annotations[addRcmDataAnnotation], obj.Annotations[addRcmBinaryDataAnnotation])
			myLog.Info(msg)

			cm.Data = obj.Data
			cm.BinaryData = obj.BinaryData

			cm.Annotations[addRcmDataAnnotation] = obj.Annotations[addRcmDataAnnotation]
			cm.Annotations[addRcmBinaryDataAnnotation] = obj.Annotations[addRcmBinaryDataAnnotation]

			err := r.Update(ctx, cm)

			if err != nil {
				obj.Status.Phase = replicationsv1alpha1.PhaseFailed
				myLog.Error(err, "unable to update configmap")
				r.Status().Update(ctx, obj)
				return ctrl.Result{}, err
			}
			//}
		}

		obj.Status.MatchingNamespaces = obj.Status.MatchingNamespaces + " " + namespace

	}
	obj.Status.Phase = replicationsv1alpha1.PhaseCompleted
	if err := r.Status().Update(ctx, obj); err != nil {
		myLog.Error(err, "unable to update final status of object")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: defaultRequeue}, nil

}

func (r *ReplicatedConfigMapReconciler) doDeleteResources(ctx context.Context, obj *replicationsv1alpha1.ReplicatedConfigMap) error {

	myLog := log.FromContext(ctx)

	msg := fmt.Sprintf("Looking for child resources of %q for deletion", obj.GetName())
	myLog.Info(msg)

	for _, namespace := range r.findNamespaces(ctx, obj) {
		// delete ConfigMap
		cm := r.getCm(ctx, obj, namespace)
		if cm != nil {
			err := r.Delete(ctx, cm)
			if err != nil {
				msg := fmt.Sprintf("Cannot delete config map %q on namespace %q. Potential misconfiguration or deletion", cm.GetName(), namespace)
				myLog.Info(msg)
				myLog.Error(err, "unable to delete child resources")
				return err
			}
		}

	}

	return nil
}

func (r *ReplicatedConfigMapReconciler) findNamespaces(ctx context.Context, obj *replicationsv1alpha1.ReplicatedConfigMap) []string {

	myLog := log.FromContext(ctx)

	nsObj := &corev1.NamespaceList{}

	err := r.List(ctx, nsObj, client.MatchingLabels{rcmLabel: "true"})

	if err != nil {
		myLog.Error(err, "unable to retrieve namespaces")
		return nil
	}
	var ns []string
	for _, item := range nsObj.Items {
		ns = append(ns, item.Name)
	}

	return ns
}

func (r *ReplicatedConfigMapReconciler) getCm(ctx context.Context, obj *replicationsv1alpha1.ReplicatedConfigMap, ns string) *corev1.ConfigMap {

	myLog := log.FromContext(ctx)

	cm := &corev1.ConfigMap{}

	err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: obj.GetName()}, cm)

	if err != nil {
		msg := fmt.Sprintf("Cannot retrieve config map %q on namespace %q", obj.GetName(), ns)
		myLog.Info(msg)
		return nil
	} else {
		return cm
	}

}

func (r *ReplicatedConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {

	c, err := controller.New("rcm-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch rcm objects
	src := &source.Kind{Type: &replicationsv1alpha1.ReplicatedConfigMap{}}
	h := &handler.EnqueueRequestForObject{}
	p := predicate.GenerationChangedPredicate{}
	if err := c.Watch(src, h, p); err != nil {
		return err
	}

	// Watch configmap objects
	errCm := c.Watch(
		&source.Kind{Type: &corev1.ConfigMap{}},
		handler.EnqueueRequestsFromMapFunc(r.cmToRequest),
		predicate.Or(predicate.ResourceVersionChangedPredicate{}),
	)

	if errCm != nil {
		return err
	}

	errNs := c.Watch(
		&source.Kind{Type: &corev1.Namespace{}},
		handler.EnqueueRequestsFromMapFunc(r.nsToRequest),
		predicate.Or(
			//predicate.LabelSelectorPredicate(rcmLabel, "true")),
			predicate.LabelChangedPredicate{}),
	)

	if errNs != nil {
		return err
	}

	return nil

}

func (r *ReplicatedConfigMapReconciler) cmToRequest(obj client.Object) []ctrl.Request {

	cms := &corev1.ConfigMapList{}

	err := r.List(context.TODO(), cms, client.MatchingLabels{rcmLabel: "true"})

	if err != nil {
		return []ctrl.Request{}
	}

	requests := make([]ctrl.Request, len(cms.Items))

	for i, item := range cms.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func (r *ReplicatedConfigMapReconciler) nsToRequest(obj client.Object) []ctrl.Request {

	// will trigger reconciliation for all rcm objects
	rcmList := &replicationsv1alpha1.ReplicatedConfigMapList{}

	err := r.List(context.TODO(), rcmList)

	if err != nil {
		return []ctrl.Request{}
	}

	requests := make([]ctrl.Request, len(rcmList.Items))

	for i, item := range rcmList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
