package kog

import (
	"context"
	iofogv1 "github.com/eclipse-iofog/iofog-operator/pkg/apis/iofog/v1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	b64 "encoding/base64"
)

var log = logf.Log.WithName("controller_kog")

// Add creates a new Kog Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKog{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kog-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Kog
	err = c.Watch(&source.Kind{Type: &iofogv1.Kog{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Kog
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iofogv1.Kog{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKog implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKog{}

// ReconcileKog reconciles a Kog object
type ReconcileKog struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	logger      logr.Logger
	apiEndpoint string
}

// Reconcile reads that state of the cluster for a Kog object and makes changes based on the state read
// and what is in the Kog.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKog) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.logger.Info("Reconciling Control Plane")

	// Fetch the Kog kog
	kog := &iofogv1.Kog{}
	err := r.client.Get(context.TODO(), request.NamespacedName, kog)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Decode credentials
	if kog.Spec.ControlPlane.IofogUser.Password, err = decode(kog.Spec.ControlPlane.IofogUser.Password); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile Iofog Controller
	if err = r.reconcileIofogController(kog); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile Iofog Kubelet
	if err = r.reconcileIofogKubelet(kog); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile Port Manager
	if err = r.reconcilePortManager(kog); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile Skupper
	if err = r.reconcileSkupper(kog); err != nil {
		return reconcile.Result{}, err
	}

	r.logger.Info("Completed Reconciliation")

	return reconcile.Result{}, nil
}

func decode(encoded string) (string, error) {
	decodedBytes, err := b64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}
