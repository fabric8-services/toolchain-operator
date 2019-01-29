package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"

	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestTooChainEnabler(t *testing.T) {

	toolChainEnablerList := &codereadyv1alpha1.ToolChainEnablerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ToolChainEnabler",
			APIVersion: "codeready.io/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, toolChainEnablerList)
	require.NoError(t, err, "failed to add custom resource scheme to framework: %v", err)

	// run subtests
	t.Run("Toolchain", func(t *testing.T) {
		t.Run("Enable", EnableCodeReadyToolChain)
	})
}

func verifyResources(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
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
	err = f.Client.Create(goctx.TODO(), exampleToolChainEnabler, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = waitForSelfProvisioningServiceAccount(t, f.KubeClient, namespace, retryInterval, timeout)
	if err != nil {
		return err
	}

	return nil
}

func EnableCodeReadyToolChain(t *testing.T) {
	os.Setenv("TEST_NAMESPACE", "toolchain-e2e-test")
	defer os.Unsetenv("TEST_NAMESPACE")
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	require.NoError(t, err, "failed to initialize cluster resources: %v", err)
	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "failed to get namespace where operator needs to run: %v", err)

	// get global framework variables
	f := framework.Global

	// wait for toolchain-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, toolchainenabler.Name, 1, retryInterval, timeout)
	require.NoError(t, err, "failed while waiting for operator deployment")

	t.Log("Toolchain operator is ready and running state")

	err = verifyResources(t, f, ctx)
	assert.NoError(t, err)
}
