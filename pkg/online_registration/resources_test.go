package online_registration

import (
	"errors"
	"fmt"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	errs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestResourceCreator(t *testing.T) {

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())
			//when
			err := EnsureServiceAccount(cl, test.NewFakeCache(errs.NewNotFound(schema.GroupResource{}, ServiceAccountName)))
			//then
			require.NoError(t, err, "failed to create SA %s", ServiceAccountName)
			assertSA(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())

			//create SA first time
			err := EnsureServiceAccount(cl, test.NewFakeCache(errs.NewNotFound(schema.GroupResource{}, ServiceAccountName)))
			require.NoError(t, err, "failed to create SA %s", ServiceAccountName)
			assertSA(t, cl)

			//when
			err = EnsureServiceAccount(cl, &test.FakeCache{})

			//then
			require.NoError(t, err, "failed to ensure SA %s", ServiceAccountName)
			assertSA(t, cl)

		})

		t.Run("fail", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())

			//when
			err := EnsureServiceAccount(cl, test.NewFakeCache(errors.New("something went wrong")))

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get service account %s from namespace %s: %s", ServiceAccountName, Namespace, "something went wrong"))
		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())

			//when
			err := EnsureClusterRoleBinding(cl)

			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())

			//when
			err := EnsureClusterRoleBinding(cl)

			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)

			// when
			err = EnsureClusterRoleBinding(cl)

			require.NoError(t, err, "failed to ensure ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting clusterrolebinding"
			m := make(map[string]string)
			m["crb"] = errMsg

			cl := test.NewDummyClient(client.NewClient(fake.NewFakeClient()), m)

			//when
			err := EnsureClusterRoleBinding(cl)

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get clusterrolebinding %s: %s", ClusterRoleBindingName, errMsg))
		})
	})
}

func assertSA(t *testing.T, cl client.Client) {
	// Check if service account has been created
	sa, err := cl.GetServiceAccount(Namespace, ServiceAccountName)
	assert.NoError(t, err, "couldn't find created sa %s in namespace %s", ServiceAccountName, Namespace)
	assert.NotNil(t, sa)
}

func assertClusterRoleBinding(t *testing.T, cl client.Client) {
	// Check service account has online-registration clusterrole
	actual, err := cl.GetClusterRoleBinding(ClusterRoleBindingName)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", ClusterRoleBindingName)
	assert.NotNil(t, actual)

	subs := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      ServiceAccountName,
			Namespace: Namespace,
		},
	}

	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "online-registration",
	}

	assert.Equal(t, actual.Subjects, subs)
	assert.Equal(t, actual.RoleRef, roleRef)
}
