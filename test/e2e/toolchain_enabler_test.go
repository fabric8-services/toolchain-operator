package e2e

import (
	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	"testing"

	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
	"github.com/fabric8-services/toolchain-operator/pkg/online_registration"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToolChainEnabler(t *testing.T) {

	toolChainEnablerList := &codereadyv1alpha1.ToolChainEnablerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ToolChainEnabler",
			APIVersion: "codeready.openshift.io/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, toolChainEnablerList)
	require.NoError(t, err, "failed to add custom resource scheme to framework")

	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	require.NoError(t, err, "failed to initialize cluster resources")
	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "failed to get namespace where operator needs to run")

	// get global framework variables
	f := framework.Global

	// wait for toolchain-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, config.Name, 1, retryInterval, timeout)
	require.NoError(t, err, "failed while waiting for operator deployment")

	t.Log("Toolchain operator is ready and running state")

	// create ToolChainEnabler custom resource
	exampleToolChainEnabler := &codereadyv1alpha1.ToolChainEnabler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ToolChainEnabler",
			APIVersion: "codeready.openshift.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-toolchainenabler",
			Namespace: namespace,
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(context.TODO(), exampleToolChainEnabler, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	require.NoError(t, err, "failed to create custom resource of kind `ToolChainEnabler`")

	operatorClient := client.NewClient(f.Client.Client)

	t.Run("verify", func(t *testing.T) {
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete oauth client and verify", func(t *testing.T) {
		// given
		oc, err := operatorClient.GetOAuthClient(config.OAuthClientName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), oc)
		require.NoError(t, err, "failed to delete oauth client %s", config.OAuthClientName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete self-provisioner cluster role binding and verify", func(t *testing.T) {
		// given
		clusterRoleBinding, err := operatorClient.GetClusterRoleBinding(toolchainenabler.SelfProvisioner)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), clusterRoleBinding)
		require.NoError(t, err, "failed to delete cluster role binding %s", toolchainenabler.SelfProvisioner)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete dsaas-cluster-admin cluster role binding and verify", func(t *testing.T) {
		// given
		clusterRoleBinding, err := operatorClient.GetClusterRoleBinding(toolchainenabler.DsaasClusterAdmin)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), clusterRoleBinding)
		require.NoError(t, err, "failed to delete cluster role binding %s", toolchainenabler.DsaasClusterAdmin)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete sa and verify", func(t *testing.T) {
		// given
		sa, err := operatorClient.GetServiceAccount(namespace, config.SAName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), sa)
		require.NoError(t, err, "failed to delete service account %s/%s", namespace, config.SAName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete online-registration cluster role binding and verify", func(t *testing.T) {
		// given
		clusterRoleBinding, err := operatorClient.GetClusterRoleBinding(online_registration.ClusterRoleBindingName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), clusterRoleBinding)
		require.NoError(t, err, "failed to delete cluster role binding %s", online_registration.ClusterRoleBindingName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete online-registration sa and verify", func(t *testing.T) {
		// given
		sa, err := operatorClient.GetServiceAccount(online_registration.Namespace, online_registration.ServiceAccountName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), sa)
		require.NoError(t, err, "failed to delete service account %s/%s", online_registration.Namespace, online_registration.ServiceAccountName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

}

func verifyResources(t *testing.T, operatorClient client.Client, namespace string) error {
	if err := waitForServiceAccount(t, operatorClient, namespace, config.SAName); err != nil {
		return err
	}

	if err := waitForClusterRoleBinding(t, operatorClient, toolchainenabler.SelfProvisioner); err != nil {
		return err
	}

	if err := waitForClusterRoleBinding(t, operatorClient, toolchainenabler.DsaasClusterAdmin); err != nil {
		return err
	}

	if err := waitForOauthClient(t, operatorClient); err != nil {
		return err
	}

	if err := waitForServiceAccount(t, operatorClient, online_registration.Namespace, online_registration.ServiceAccountName); err != nil {
		return err
	}

	if err := waitForClusterRoleBinding(t, operatorClient, online_registration.ClusterRoleBindingName); err != nil {
		return err
	}

	return nil
}
