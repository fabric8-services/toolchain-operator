package e2e

import (
	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	"testing"

	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func TestToolChainEnabler(t *testing.T) {

	toolChainEnablerList := &codereadyv1alpha1.ToolChainEnablerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ToolChainEnabler",
			APIVersion: "codeready.io/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, toolChainEnablerList)
	require.NoError(t, err, "failed to add custom resource scheme to framework")

	os.Setenv("TEST_NAMESPACE", "toolchain-e2e-test")
	defer os.Unsetenv("TEST_NAMESPACE")
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
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, toolchainenabler.Name, 1, retryInterval, timeout)
	require.NoError(t, err, "failed while waiting for operator deployment")

	t.Log("Toolchain operator is ready and running state")

	// create ToolChainEnabler custom resource
	exampleToolChainEnabler := &codereadyv1alpha1.ToolChainEnabler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ToolChainEnabler",
			APIVersion: "codeready.io/v1alpha1",
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
		oc, err := operatorClient.GetOAuthClient(toolchainenabler.OAuthClientName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), oc)
		require.NoError(t, err, "failed to delete oauth client %s", toolchainenabler.OAuthClientName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete cluster role binding and verify", func(t *testing.T) {
		// given
		clusterRoleBinding, err := operatorClient.GetClusterRoleBinding(toolchainenabler.CRBName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), clusterRoleBinding)
		require.NoError(t, err, "failed to delete cluster role binding %s", toolchainenabler.CRBName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})

	t.Run("delete sa and verify", func(t *testing.T) {
		// given
		sa, err := operatorClient.GetServiceAccount(namespace, toolchainenabler.SAName)
		require.NoError(t, err)

		// when
		err = operatorClient.Delete(context.Background(), sa)
		require.NoError(t, err, "failed to delete service account %s/%s", namespace, toolchainenabler.SAName)

		// then
		err = verifyResources(t, operatorClient, namespace)
		assert.NoError(t, err)
	})
}

func verifyResources(t *testing.T, operatorClient client.Client, namespace string) error {
	if err := waitForServiceAccount(t, operatorClient, namespace); err != nil {
		return err
	}

	if err := waitForClusterRoleBinding(t, operatorClient); err != nil {
		return err
	}

	if err := waitForOauthClient(t, operatorClient); err != nil {
		return err
	}

	return nil
}
