package toolchainenabler

import (
	"context"

	"fmt"
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_toolchainenabler")

const (
	Name    = "toolchain-enabler"
	SAName  = "toolchain-sre"
	CRBName = "system:toolchain-enabler:self-provisioner"
)

// Add creates a new ToolChainEnabler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileToolChainEnabler{client: client.NewClient(mgr.GetClient()), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("toolchainenabler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ToolChainEnabler
	if err := c.Watch(&source.Kind{Type: &codereadyv1alpha1.ToolChainEnabler{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Watch for changes to secondary resource Service Account and requeue the owner ToolChainEnabler
	enqueueRequestForOwner := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codereadyv1alpha1.ToolChainEnabler{},
	}

	if err := c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileToolChainEnabler{}

// ReconcileToolChainEnabler reconciles a ToolChainEnabler object
type ReconcileToolChainEnabler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ToolChainEnabler object and makes changes based on the state read
// and what is in the ToolChainEnabler.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileToolChainEnabler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ToolChainEnabler")

	// Fetch the ToolChainEnabler instance
	instance := &codereadyv1alpha1.ToolChainEnabler{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Create SA
	if err := r.ensureSA(instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.ensureClusterRoleBinding(instance, SAName, instance.Namespace); err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Skip reconcile: as Service Account 'toolchain-sre' created with self-provisioner cluster role")
	return reconcile.Result{}, nil
}

func (r *ReconcileToolChainEnabler) ensureSA(tce *codereadyv1alpha1.ToolChainEnabler) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SAName,
			Namespace: tce.Namespace,
		},
	}

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, sa, r.scheme); err != nil {
		return err
	}

	_, err := r.client.GetServiceAccount(tce.Namespace, SAName)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service Account ", "Namespace", sa.Namespace, "Name", sa.Name)
		if err = r.client.CreateServiceAccount(sa); err != nil {
			return err
		}

		// SA created successfully
		return nil
	}
	log.Info(fmt.Sprintf("ServiceAccount `%s` already exists", SAName))

	return nil
}

// create ClusterRoleBinding for Service Account with self-provisioner Role
func (r *ReconcileToolChainEnabler) ensureClusterRoleBinding(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	crb := &rbacv1.ClusterRoleBinding{
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "self-provisioner",
		},
	}

	crb.SetName(CRBName)

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, crb, r.scheme); err != nil {
		return err
	}
	_, err := r.client.GetClusterRoleBinding(CRBName)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Adding `self-provisioner` cluster role to ", "Service Account", saName)
		if err := r.client.CreateClusterRoleBinding(crb); err != nil {
			return err
		}

		// ClusterRoleBinding created successfully
		return nil
	}

	log.Info(fmt.Sprintf("ClusterRoleBinding `%s` already exists", CRBName))

	return nil
}
