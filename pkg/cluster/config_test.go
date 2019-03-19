package cluster

import (
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/fabric8-services/toolchain-operator/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigOption(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		// given
		cl := client.NewClient(fake.NewFakeClient())
		informer := configInformer{cl, "test-configInformer", "test-cluster"}

		clusterData := &clusterclient.CreateClusterData{}
		nameOption := name(informer)

		// when
		err := nameOption(clusterData)
		require.NoError(t, err)

		// then
		assert.Equal(t, clusterData.Name, "test-cluster")
	})

	t.Run("oauth client", func(t *testing.T) {
		// given
		err := apis.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		cl := client.NewClient(fake.NewFakeClient())

		var ageSeconds int32
		oc := &oauthv1.OAuthClient{
			ObjectMeta: metav1.ObjectMeta{
				Name: config.OAuthClientName,
			},
			Secret:                   "oauthsecret",
			GrantMethod:              oauthv1.GrantHandlerAuto,
			RedirectURIs:             []string{"https://auth.openshift.io/"},
			AccessTokenMaxAgeSeconds: &ageSeconds,
		}

		err = cl.CreateOAuthClient(oc)
		require.NoError(t, err)

		informer := configInformer{cl, "test-configInformer", "test-cluster"}

		clusterData := &clusterclient.CreateClusterData{}
		OauthClientOption := oauthClient(informer)

		// when
		err = OauthClientOption(clusterData)
		require.NoError(t, err)

		// then
		assert.Equal(t, clusterData.AuthClientID, "codeready-toolchain")
		assert.Equal(t, clusterData.AuthClientSecret, "oauthsecret")
		assert.Equal(t, clusterData.AuthClientDefaultScope, "user:full")
	})

	t.Run("sa", func(t *testing.T) {
		t.Run("secret ref", func(t *testing.T) {
			// given
			cl := client.NewClient(fake.NewFakeClient())
			ns := "config-test"
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.SAName,
					Namespace: ns,
				},
			}
			err := cl.CreateServiceAccount(sa)
			require.NoError(t, err)

			// create secrets for sa as we are using fake client
			saSecretOptions := test.SASecretOption(t, cl, ns)
			informer := configInformer{cl, ns, "test-cluster"}

			clusterData := &clusterclient.CreateClusterData{}
			SAOption := serviceAccount(informer, saSecretOptions)

			// when
			err = SAOption(clusterData)
			require.NoError(t, err)

			// then
			assert.Equal(t, clusterData.ServiceAccountUsername, "system:serviceaccount:config-test:toolchain-sre")
			assert.Equal(t, clusterData.ServiceAccountToken, "mysatoken")
		})

		t.Run("no secret ref", func(t *testing.T) {
			// given
			ns := "config-test"
			cl := client.NewClient(fake.NewFakeClient())

			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.SAName,
					Namespace: ns,
				},
			}
			err := cl.CreateServiceAccount(sa)
			require.NoError(t, err)

			informer := configInformer{cl, ns, "test-cluster"}

			clusterData := &clusterclient.CreateClusterData{}
			SAOption := serviceAccount(informer)

			// when
			err = SAOption(clusterData)

			// then
			assert.EqualError(t, err, "couldn't find any secret reference for sa toolchain-sre")
		})

		t.Run("no secret reference of type 'kubernetes.io/service-account-token'", func(t *testing.T) {
			// given
			cl := client.NewClient(fake.NewFakeClient())

			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.SAName,
					Namespace: "config-test",
				},
			}
			err := cl.CreateServiceAccount(sa)
			require.NoError(t, err)

			// create secrets for sa as we are using fake client
			err = cl.CreateSecret(test.Secret("toolchain-sre-6756s", "config-test", "mydockertoken", corev1.SecretTypeDockercfg))
			require.NoError(t, err)

			informer := configInformer{cl, "config-test", "test-cluster"}

			clusterData := &clusterclient.CreateClusterData{}
			SAOption := serviceAccount(informer, func(sa *corev1.ServiceAccount) {
				sa.Secrets = append(sa.Secrets,
					corev1.ObjectReference{Name: "toolchain-sre-6756s", Namespace: "config-test", Kind: "Secret"},
				)
			})

			// when
			err = SAOption(clusterData)

			// then
			assert.EqualError(t, err, "couldn't find any secret reference for sa toolchain-sre of type kubernetes.io/service-account-token")
		})

	})

	t.Run("cluster url", func(t *testing.T) {
		// given
		cl := client.NewClient(fake.NewFakeClient())
		informer := configInformer{cl, "test-configInformer", "test-cluster"}
		clusterData := &clusterclient.CreateClusterData{}
		urlOption := apiURL(informer)

		// when
		err := urlOption(clusterData)
		require.NoError(t, err)

		// then
		assert.Equal(t, clusterData.APIURL, "https://api.test-cluster.openshift.com/")
	})

	t.Run("app dns", func(t *testing.T) {
		// given
		err := apis.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		cl := client.NewClient(fake.NewFakeClient())
		informer := configInformer{cl, "test-configInformer", "test-cluster"}
		clusterData := &clusterclient.CreateClusterData{}
		appDNSOption := appDNS(informer, withRouteHost("foo-dipakpawar231.8a09.starter-us-east-2.openshiftapps.com"))

		// when
		err = appDNSOption(clusterData)
		require.NoError(t, err)

		// then
		assert.Equal(t, clusterData.AppDNS, "8a09.starter-us-east-2.openshiftapps.com")
	})

	t.Run("token provider", func(t *testing.T) {
		// given
		clusterData := &clusterclient.CreateClusterData{}
		tokenProviderOption := tokenProvider()

		// when
		err := tokenProviderOption(clusterData)
		require.NoError(t, err)

		// then
		require.NotNil(t, clusterData.TokenProviderID)
		assert.NotEmpty(t, len(*clusterData.TokenProviderID))
	})

	t.Run("type osd", func(t *testing.T) {
		// given
		clusterData := &clusterclient.CreateClusterData{}
		typeOSDOption := typeOSD()

		// when
		err := typeOSDOption(clusterData)
		require.NoError(t, err)

		// then
		assert.Equal(t, "OSD", clusterData.Type)
	})
}
