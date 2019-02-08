package toolchainenabler

import (
	"context"

	"fmt"
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	oauthv1 "github.com/openshift/api/oauth/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/fabric8-services/toolchain-operator/pkg/secret"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	Name            = "toolchain-enabler"
	SAName          = "toolchain-sre"
	OAuthClientName = "codeready-toolchain"
	CRBName         = "system:toolchain-enabler:self-provisioner"
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

	return c.Watch(&source.Kind{Type: &oauthv1.OAuthClient{}}, enqueueRequestForOwner)
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
	namespacedName := request.NamespacedName

	// overwrite for cluster scoped resources like OAuthClient, ClusterRoleBinding as you can't get namespace from it's event
	if request.Namespace == "" {
		log.Info("Couldn't find namespace in the request, getting it from env variable `WATCH_NAMESPACE`")
		ns, err := k8sutil.GetWatchNamespace()
		if err != nil {
			log.Error(err, "can't reconcile request coming from cluster scoped resources event")
			return reconcile.Result{}, nil
		}
		namespacedName = types.NamespacedName{Namespace: ns, Name: request.Name}
	}
	if err := r.client.Get(context.TODO(), namespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Requeueing request doesn't start as couldn't find requested object or stopped as requested object could have been deleted")
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

	if err := r.ensureOAuthClient(instance); err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Skipping reconcile as all required objects are created and exist", "Service Account", SAName, "ClusterRoleBindning", CRBName, "OAuthClient", OAuthClientName)
	return reconcile.Result{}, nil
}

// ensureSA creates Service Account if not exists
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

		log.Info(fmt.Sprintf("ServiceAccount `%s` created successfully", SAName))
		return nil
	}
	log.Info(fmt.Sprintf("ServiceAccount `%s` already exists", SAName))

	return nil
}

// ensureClusterRoleBinding ensures ClusterRoleBinding for Service Account with self-provisioner Role
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

		log.Info(fmt.Sprintf("ClusterRoleBinding %s created successfully", CRBName))
		return nil
	}

	log.Info(fmt.Sprintf("ClusterRoleBinding `%s` already exists", CRBName))

	return nil
}

// ensureOAuthClient creates OAuthClient if not exists
func (r *ReconcileToolChainEnabler) ensureOAuthClient(tce *codereadyv1alpha1.ToolChainEnabler) error {
	randomString, err := secret.CreateRandomString(256)
	if err != nil {
		return errs.Wrapf(err, "failed to generate random string to be used as secret for OAuthClient")
	}
	var ageSeconds int32
	oc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: OAuthClientName,
		},
		Secret:                   randomString,
		GrantMethod:              oauthv1.GrantHandlerAuto,
		RedirectURIs:             []string{"https://auth.openshift.io/"},
		AccessTokenMaxAgeSeconds: &ageSeconds,
	}

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, oc, r.scheme); err != nil {
		return err
	}
	_, err = r.client.GetOAuthClient(OAuthClientName)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating", "OAuthClient", OAuthClientName)
		if err := r.client.CreateOAuthClient(oc); err != nil {
			return err
		}

		log.Info(fmt.Sprintf("OAuthClient %s created successfully", OAuthClientName))
		return nil
	}

	log.Info(fmt.Sprintf("OAuthClient `%s` already exists", OAuthClientName))

	return nil
}
