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

	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/toolchain-operator/pkg/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
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
	"time"
)

var log = logf.Log.WithName("controller_toolchainenabler")

const (
	SelfProvisioner   = "system:toolchain-sre:self-provisioner"
	DsaasClusterAdmin = "system:toolchain-sre:dsaas-cluster-admin"
)

// Add creates a new ToolChainEnabler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	c, err := config.NewConfiguration()
	if err != nil {
		return nil, errs.Wrapf(err, "something went wrong while creating configuration")
	}
	return &ReconcileToolChainEnabler{client: client.NewClient(mgr.GetClient()), scheme: mgr.GetScheme(), config: c}, nil
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
	config *config.Configuration
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
		log.Info(`couldn't find namespace in the request, getting it from env variable "WATCH_NAMESPACE"`)
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

	if err := r.ensureClusterRoleBinding(instance, config.SAName, instance.Namespace); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.ensureOAuthClient(instance); err != nil {
		return reconcile.Result{}, err
	}

	clusterData, err := r.clusterInfo(namespacedName.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.saveClusterConfiguration(clusterData); err != nil {
		log.Error(err, "failed to save cluster configuration in cluster service", "cluster_service_url", r.config.GetClusterServiceURL())
		// requeue after 5 seconds if failed while calling remote cluster service
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	reqLogger.Info("Skipping reconcile as cluster configuration has been updated to cluster management service successfully")
	return reconcile.Result{}, nil
}

// ensureSA creates Service Account if not exists
func (r ReconcileToolChainEnabler) ensureSA(tce *codereadyv1alpha1.ToolChainEnabler) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SAName,
			Namespace: tce.Namespace,
		},
	}

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, sa, r.scheme); err != nil {
		return err
	}

	if _, err := r.client.GetServiceAccount(tce.Namespace, config.SAName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating a new service sccount ", "namespace", sa.Namespace, "name", sa.Name)
			if err := r.client.CreateServiceAccount(sa); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("service account %s created successfully", config.SAName))
			return nil
		}
		return errs.Wrapf(err, "failed to get service account %s", config.SAName)
	}
	log.Info(fmt.Sprintf("service account %s already exists", config.SAName))

	return nil
}

// ensureClusterRoleBinding ensures ClusterRoleBinding for Service Account with required roles
func (r *ReconcileToolChainEnabler) ensureClusterRoleBinding(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	if err := r.bindSelfProvisionerRole(tce, saName, namespace); err != nil {
		return err
	}

	return r.bindDsaasClusterAdminRole(tce, saName, namespace)
}

// bindSelfProvisionerRole creates ClusterRoleBinding for Service Account with self-provisioner cluster role
func (r *ReconcileToolChainEnabler) bindSelfProvisionerRole(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
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

	crb.SetName(SelfProvisioner)

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, crb, r.scheme); err != nil {
		return err
	}
	if _, err := r.client.GetClusterRoleBinding(SelfProvisioner); err != nil {
		if errors.IsNotFound(err) {
			log.Info(`adding "self-provisioner" cluster role to `, "Service Account", saName)
			if err := r.client.CreateClusterRoleBinding(crb); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", SelfProvisioner))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", SelfProvisioner)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", SelfProvisioner))

	return nil
}

// bindDsaasClusterAdminRole creates ClusterRoleBinding for Service Account with dsaas-cluster-admin cluster role
func (r *ReconcileToolChainEnabler) bindDsaasClusterAdminRole(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	// currently we have defined ClusterRole dsaas-cluster-admin which needs to be create before running this operator.
	// TODO: we should verify this cluster role existence and create if missing.
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
			Name:     "dsaas-cluster-admin",
		},
	}

	crb.SetName(DsaasClusterAdmin)

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, crb, r.scheme); err != nil {
		return err
	}
	if _, err := r.client.GetClusterRoleBinding(DsaasClusterAdmin); err != nil {
		if errors.IsNotFound(err) {
			log.Info(`adding "dsaas-cluster-admin" cluster role to `, "Service Account", saName)
			if err := r.client.CreateClusterRoleBinding(crb); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", DsaasClusterAdmin))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", DsaasClusterAdmin)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", DsaasClusterAdmin))

	return nil
}

// ensureOAuthClient creates OAuthClient if not exists
func (r ReconcileToolChainEnabler) ensureOAuthClient(tce *codereadyv1alpha1.ToolChainEnabler) error {
	randomString, err := secret.CreateRandomString(256)
	if err != nil {
		return errs.Wrapf(err, "failed to generate random string to be used as secret for oauthclient")
	}
	var ageSeconds int32
	oc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.OAuthClientName,
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

	if _, err = r.client.GetOAuthClient(config.OAuthClientName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating", "oauthclient", config.OAuthClientName)
			if err := r.client.CreateOAuthClient(oc); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("oauth client %s created successfully", config.OAuthClientName))
			return nil
		}
		return errs.Wrapf(err, "failed to get oauthclient %s", config.OAuthClientName)
	}

	log.Info(fmt.Sprintf("oauth client %s already exists", config.OAuthClientName))

	return nil
}

func (r ReconcileToolChainEnabler) clusterInfo(ns string, options ...cluster.SASecretOption) (*clusterclient.CreateClusterData, error) {
	i := cluster.NewConfigInformer(r.client, ns, r.config.GetClusterName())
	return i.Inform(options...)
}

func (r ReconcileToolChainEnabler) saveClusterConfiguration(data *clusterclient.CreateClusterData, options ...httpsupport.HTTPClientOption) error {
	service := cluster.NewClusterService(r.config)
	return service.CreateCluster(context.Background(), data, options...)
}
