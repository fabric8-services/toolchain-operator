package cluster

import (
	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/magiconair/properties/assert"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestRoutingSubDomain(t *testing.T) {
	// given
	err := apis.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	cl := client.NewClient(fake.NewFakeClient())
	informer := NewInformer(cl, "test-informer")

	// when
	sd, err := informer.routingSubDomain(withRouteHost("foo-dipakpawar231.8a09.starter-us-east-2.openshiftapps.com"))

	//then
	require.NoError(t, err)
	_, err = cl.GetRoute("test-informer", RouteName)
	require.Error(t, err, "couldn't delete route")
	require.EqualError(t, err, "routes.route.openshift.io \"toolchain-route\" not found")

	assert.Equal(t, sd, "8a09.starter-us-east-2.openshiftapps.com")
}

func TestRouteHostSuffix(t *testing.T) {

	t.Run("host with protocol", func(t *testing.T) {
		// given
		host := "https://foo-dipakpawar231.8a09.starter-us-east-2.openshiftapps.com"

		// when
		subDomain := routeHostSubDomain(host)

		// then
		assert.Equal(t, subDomain, "8a09.starter-us-east-2.openshiftapps.com")
	})

	t.Run("host without protocol", func(t *testing.T) {
		// given
		host := "foo-dipakpawar231.8a09.starter-us-east-2.openshiftapps.com"

		// when
		subDomain := routeHostSubDomain(host)

		// then
		assert.Equal(t, subDomain, "8a09.starter-us-east-2.openshiftapps.com")
	})

	t.Run("host with space", func(t *testing.T) {
		// given
		host := "foo-dipakpawar231.8a09.starter-us-east-2.openshiftapps.com "

		// when
		subDomain := routeHostSubDomain(host)

		// then
		assert.Equal(t, subDomain, "8a09.starter-us-east-2.openshiftapps.com")
	})

	t.Run("empty", func(t *testing.T) {
		// given
		host := ""

		// when
		subDomain := routeHostSubDomain(host)

		// then
		assert.Equal(t, subDomain, "")
	})
}

func withRouteHost(h string) RouteOption {
	return func(route *routev1.Route) {
		route.Spec.Host = h
	}
}
