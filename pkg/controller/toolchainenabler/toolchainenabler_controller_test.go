package toolchainenabler

import (
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"context"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	Namespace = "codeready-toolchain"
)

// TestToolChainEnablerController runs ReconcileToolChainEnabler.Reconcile() against a
// fake client that tracks a ToolChainEnabler object.
func TestToolChainEnablerController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	// A ToolChainEnabler resource with metadata and spec.
	tce := &codereadyv1alpha1.ToolChainEnabler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
		},
		Spec: codereadyv1alpha1.ToolChainEnablerSpec{},
	}
	// Objects to track in the fake client.
	objs := []runtime.Object{
		tce,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(codereadyv1alpha1.SchemeGroupVersion, tce)

	t.Run("Reconcile", func(t *testing.T) {
		t.Run("With ToolChainEnabler Custom Resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			assertSA(t, cl)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("without ToolChainEnabler Custom Resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls without any runtime object
			cl := client.NewClient(fake.NewFakeClient())

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			sa, err := cl.GetServiceAccount(Namespace, SAName)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, sa, "found sa %s", SAName)

			actual, err := cl.GetClusterRoleBinding(CRBName)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, actual, "found ClusterRoleBinding %s", CRBName)
		})

	})

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureSA(instance)
			//then
			require.NoError(t, err, "failed to create SA %s", SAName)
			assertSA(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//create SA first time
			err = r.ensureSA(instance)
			require.NoError(t, err, "failed to create SA %s", SAName)
			assertSA(t, cl)

			//when
			err = r.ensureSA(instance)

			//then
			require.NoError(t, err, "failed to ensure SA %s", SAName)
			assertSA(t, cl)

		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)
			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", SAName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create ClusterRolebinding first time
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to create ClusterRoleBinding %s", CRBName)
			assertClusterRoleBinding(t, cl)

			// when
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to ensure ClusterRoleBinding %s", CRBName)
			assertClusterRoleBinding(t, cl)
		})
	})

	t.Run("OAuthClient", func(t *testing.T) {
		// register openshift resource OAuthClient specific schema
		err := oauthv1.Install(s)
		require.NoError(t, err)

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureOAuthClient(instance)
			//then
			require.NoError(t, err, "failed to create OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create OAuthClient first time
			err = r.ensureOAuthClient(instance)

			require.NoError(t, err, "failed to create OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)

			// when
			err = r.ensureOAuthClient(instance)

			require.NoError(t, err, "failed to ensure OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)
		})
	})
}

func assertSA(t *testing.T, cl client.Client) {
	// Check if Service Account has been created
	sa, err := cl.GetServiceAccount(Namespace, SAName)
	assert.NoError(t, err, "couldn't find created sa %s in namespace %s", SAName, Namespace)
	assert.NotNil(t, sa)
}

func assertClusterRoleBinding(t *testing.T, cl client.Client) {
	// Check Service Account has self-provision ClusterRole
	actual, err := cl.GetClusterRoleBinding(CRBName)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", CRBName)
	assert.NotNil(t, actual)

	subs := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			APIGroup:  "",
			Name:      SAName,
			Namespace: Namespace,
		},
	}
	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "self-provisioner",
	}

	assert.Equal(t, actual.Subjects, subs)
	assert.Equal(t, actual.RoleRef, roleRef)
}

func assertOAuthClient(t *testing.T, cl client.Client) {
	// Check OAuthClient has been created
	actual, err := cl.GetOAuthClient(OAuthClientName)
	assert.NoError(t, err, "couldn't find OAuthClient %s", OAuthClientName)
	assert.NotNil(t, actual)

	require.NotNil(t, actual.AccessTokenMaxAgeSeconds)
	assert.Equal(t, *actual.AccessTokenMaxAgeSeconds, int32(0))

	assert.NotEmpty(t, actual.Secret)
	assert.Equal(t, actual.GrantMethod, oauthv1.GrantHandlerAuto)
	assert.Equal(t, actual.RedirectURIs, []string{"https://auth.openshift.io/"})
}

func reconcileRequest(name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: Namespace,
		},
	}
}
