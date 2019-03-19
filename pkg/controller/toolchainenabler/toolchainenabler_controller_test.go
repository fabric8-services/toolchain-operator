package toolchainenabler

import (
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"context"
	"fmt"
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	. "github.com/fabric8-services/toolchain-operator/pkg/config"
	. "github.com/fabric8-services/toolchain-operator/test"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"os"
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
		t.Run("without registering openshift specific resources", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

			//when
			_, err := r.Reconcile(req)

			//then
			_, oautherr := cl.GetOAuthClient(OAuthClientName)
			assert.EqualError(t, err, fmt.Sprintf("failed to get oauthclient %s: %s", OAuthClientName, oautherr))
		})

		t.Run("without ToolChainEnabler custom resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls without any runtime object
			cl := client.NewClient(fake.NewFakeClient())

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			sa, err := cl.GetServiceAccount(Namespace, SAName)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, sa, "found sa %s", SAName)

			actual, err := cl.GetClusterRoleBinding(SelfProvisioner)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, actual, "found ClusterRoleBinding %s", SelfProvisioner)
		})

	})

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

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

			req := newReconcileRequest(Name)

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

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting sa"
			m := make(map[string]string)
			m["sa"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureSA(instance)

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get service account %s: %s", SAName, errMsg))
		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

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
			req := newReconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create ClusterRolebinding first time
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to create ClusterRoleBinding %s", SelfProvisioner)
			assertClusterRoleBinding(t, cl)

			// when
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to ensure ClusterRoleBinding %s", SelfProvisioner)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting clusterrolebinding"
			m := make(map[string]string)
			m["crb"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := newReconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get clusterrolebinding %s: %s", SelfProvisioner, errMsg))
		})
	})

	t.Run("OAuthClient", func(t *testing.T) {
		// register openshift resources specific schema
		err := apis.AddToScheme(scheme.Scheme)
		require.NoError(t, err)

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

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
			req := newReconcileRequest(Name)

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

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting oauthclient"
			m := make(map[string]string)
			m["oc"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := newReconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureOAuthClient(instance)
			//then
			assert.Error(t, err, "failed to get oauthclient %s: %s", OAuthClientName, errMsg)
		})
	})

	t.Run("cluster config", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			// given
			// register openshift resources specific schema
			err := apis.AddToScheme(scheme.Scheme)
			require.NoError(t, err)
			reset := SetEnv(Env("CLUSTER_NAME", "dsaas-stage"), Env("TC_CLIENT_ID", "toolchain"), Env("TC_CLIENT_SECRET", "secret"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
			defer reset()

			conf, err := NewConfiguration()
			require.NoError(t, err)

			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s, config: conf}

			// create sa, rolebinding, oauthclient resources
			req := newReconcileRequest(Name)
			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err = r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)
			require.NoError(t, err)
			assertClusterRoleBinding(t, cl)

			err = r.ensureSA(instance)
			require.NoError(t, err)
			err = r.ensureOAuthClient(instance)
			require.NoError(t, err)

			// create secrets required to refer in service account
			saSecretOption := SASecretOption(t, cl, Namespace)

			//when
			clusterData, err := r.clusterInfo(Namespace, saSecretOption)

			//then
			require.NoError(t, err, "reconcile is failing")
			assertClusterData(t, clusterData)
		})

		t.Run("fail as sa secret not present", func(t *testing.T) {
			// given
			// register openshift resources specific schema
			err := apis.AddToScheme(scheme.Scheme)
			require.NoError(t, err)
			reset := SetEnv(Env("CLUSTER_NAME", "dsaas-stage"), Env("TC_CLIENT_ID", "toolchain"), Env("TC_CLIENT_SECRET", "secret"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
			defer reset()

			conf, err := NewConfiguration()
			require.NoError(t, err)

			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s, config: conf}

			// create sa, rolebinding, oauthclient resources
			req := newReconcileRequest(Name)
			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err = r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)
			require.NoError(t, err)
			assertClusterRoleBinding(t, cl)
			err = r.ensureSA(instance)
			require.NoError(t, err)
			err = r.ensureOAuthClient(instance)
			require.NoError(t, err)

			//when
			_, err = r.clusterInfo(Namespace)

			//then
			assert.EqualError(t, err, "couldn't find any secret reference for sa toolchain-sre")
		})
	})

	t.Run("save cluster config", func(t *testing.T) {
		// given
		defer gock.OffAll()

		gock.New("http://auth").
			Post("api/token").
			MatchHeader("Content-Type", "application/x-www-form-urlencoded").
			BodyString(`client_id=bb6d043d-f243-458f-8498-2c18a12dcf47&client_secret=secret&grant_type=client_credentials`).
			Reply(200).
			BodyString(`{"access_token":"eyJhbGciOiJSUzI1NiIsImtpZCI6IjlNTG5WaWFSa2hWajFHVDlrcFdVa3dISXdVRC13WmZVeFItM0Nwa0UtWHMiLCJ0eXAiOiJKV1QifQ.eyJpYXQiOjE1NTEyNjA3NTIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3QiLCJqdGkiOiI2OTE1MmU1Mi05ZmNiLTQ3MjEtYjhlZC04MTgxY2UyOTY4ZDgiLCJzY29wZXMiOlsidW1hX3Byb3RlY3Rpb24iXSwic2VydmljZV9hY2NvdW50bmFtZSI6InRvb2xjaGFpbi1vcGVyYXRvciIsInN1YiI6ImJiNmQwNDNkLWYyNDMtNDU4Zi04NDk4LTJjMThhMTJkY2Y0NyJ9.D-t7lrfJ-nd4P62t6oXOrYY364h2yGxw23-2qoRMERdBED2E8pMAOk1yZeCk18FUn1TFslxL2nuYOE9bRL7i8qUQCGTzgFIk8QtIOw8iLSkRRPVHJGSraUSVZqsePgcU4Y_dCEZlEBkR_SPEZ5l5lm7QdfWd-JaCLnQVTW5oRPhEx0B6455UyX6Giy68ySO5WuBl0WHIvEHr6N3rSIZ7cptRAatvb9PEKxyajfBE1uC60jEE5iJwEfzv2BYBr07lhskTxQqno05In21_rRcBMjaLStVLHRVmb62hPw4FC3OGOU1wn9MmhlZVo9VYuVMjpl4qerX1l8ZkhIZpRXCpEg","token_type":"Bearer"}`)

		gock.New("http://cluster").
			Post("api/clusters").
			MatchHeader("Authorization", "Bearer "+"eyJhbGciOiJSUzI1NiIsImtpZCI6IjlNTG5WaWFSa2hWajFHVDlrcFdVa3dISXdVRC13WmZVeFItM0Nwa0UtWHMiLCJ0eXAiOiJKV1QifQ.eyJpYXQiOjE1NTEyNjA3NTIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3QiLCJqdGkiOiI2OTE1MmU1Mi05ZmNiLTQ3MjEtYjhlZC04MTgxY2UyOTY4ZDgiLCJzY29wZXMiOlsidW1hX3Byb3RlY3Rpb24iXSwic2VydmljZV9hY2NvdW50bmFtZSI6InRvb2xjaGFpbi1vcGVyYXRvciIsInN1YiI6ImJiNmQwNDNkLWYyNDMtNDU4Zi04NDk4LTJjMThhMTJkY2Y0NyJ9.D-t7lrfJ-nd4P62t6oXOrYY364h2yGxw23-2qoRMERdBED2E8pMAOk1yZeCk18FUn1TFslxL2nuYOE9bRL7i8qUQCGTzgFIk8QtIOw8iLSkRRPVHJGSraUSVZqsePgcU4Y_dCEZlEBkR_SPEZ5l5lm7QdfWd-JaCLnQVTW5oRPhEx0B6455UyX6Giy68ySO5WuBl0WHIvEHr6N3rSIZ7cptRAatvb9PEKxyajfBE1uC60jEE5iJwEfzv2BYBr07lhskTxQqno05In21_rRcBMjaLStVLHRVmb62hPw4FC3OGOU1wn9MmhlZVo9VYuVMjpl4qerX1l8ZkhIZpRXCpEg").
			BodyString(`{"data":{"api-url":"https://api.dsaas-stage.openshift.com/","app-dns":"8a09.starter-us-east-2.openshiftapps.com","auth-client-default-scope":"user:full","auth-client-id":"codeready-toolchain","auth-client-secret":"oauthsecret","name":"dsaas-stage","service-account-token":"mysatoken","service-account-username":"system:serviceaccount:config-test:toolchain-sre","token-provider-id":"3d7b75e3-7053-4846-9b64-26cf42717692","type":"OSD"}}`).
			Reply(201)

		reset := SetEnv(Env("CLUSTER_NAME", "dsaas-stage"), Env("TC_CLIENT_ID", "bb6d043d-f243-458f-8498-2c18a12dcf47"), Env("TC_CLIENT_SECRET", "secret"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
		defer reset()
		c, err := NewConfiguration()
		require.NoError(t, err)

		// Create a fake client to mock API calls.
		cl := client.NewClient(fake.NewFakeClient(objs...))

		// Create a ReconcileToolChainEnabler object with the scheme and fake client.
		r := &ReconcileToolChainEnabler{client: cl, scheme: s, config: c}

		tokenID := "3d7b75e3-7053-4846-9b64-26cf42717692"
		clusterData := &clusterclient.CreateClusterData{
			Name:                   os.Getenv("CLUSTER_NAME"),
			APIURL:                 `https://api.` + os.Getenv("CLUSTER_NAME") + `.openshift.com/`,
			AppDNS:                 "8a09.starter-us-east-2.openshiftapps.com",
			ServiceAccountToken:    "mysatoken",
			ServiceAccountUsername: "system:serviceaccount:config-test:toolchain-sre",
			AuthClientID:           "codeready-toolchain",
			AuthClientSecret:       "oauthsecret",
			AuthClientDefaultScope: "user:full",
			TokenProviderID:        &tokenID,
			Type:                   "OSD",
		}

		// when
		err = r.saveClusterConfiguration(clusterData, httpsupport.WithRoundTripper(http.DefaultTransport))

		//then
		assert.NoError(t, err)
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
	actual, err := cl.GetClusterRoleBinding(SelfProvisioner)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", SelfProvisioner)
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

	// Check Service Account has dsaas-cluster-admin ClusterRole
	dsaasClusterAdmin, err := cl.GetClusterRoleBinding(DsaasClusterAdmin)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", DsaasClusterAdmin)
	assert.NotNil(t, dsaasClusterAdmin)

	dsaasClusterAdminRoleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "dsaas-cluster-admin",
	}

	assert.Equal(t, subs, dsaasClusterAdmin.Subjects)
	assert.Equal(t, dsaasClusterAdminRoleRef, dsaasClusterAdmin.RoleRef)
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

func assertClusterData(t *testing.T, data *clusterclient.CreateClusterData) {
	require.NotNil(t, data)

	assert.Equal(t, data.Name, "dsaas-stage")
	assert.Equal(t, data.Type, "OSD")
	assert.Equal(t, data.APIURL, "https://api.dsaas-stage.openshift.com/")
	assert.Equal(t, data.AuthClientID, "codeready-toolchain")
	assert.NotEmpty(t, data.AuthClientSecret)
	assert.Equal(t, data.AuthClientDefaultScope, "user:full")
	assert.Equal(t, data.ServiceAccountUsername, "system:serviceaccount:codeready-toolchain:toolchain-sre")
	assert.Equal(t, data.ServiceAccountToken, "mysatoken")
}

func newReconcileRequest(name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: Namespace,
		},
	}
}

type DummyClient struct {
	client.Client
	resources map[string]string
}

func NewDummyClient(k8sClient client.Client, opts map[string]string) client.Client {
	return &DummyClient{k8sClient, opts}
}

func (d *DummyClient) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	if msg, ok := d.resources["sa"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetServiceAccount(namespace, name)
}

func (d *DummyClient) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	if msg, ok := d.resources["crb"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetClusterRoleBinding(name)
}

func (d *DummyClient) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	if msg, ok := d.resources["oc"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetOAuthClient(name)
}
