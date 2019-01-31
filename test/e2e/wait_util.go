package e2e

import (
	"context"
	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
	oauthv1 "github.com/openshift/api/oauth/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"reflect"
	"testing"
	"time"
)

func waitForSelfProvisioningServiceAccount(t *testing.T, kubeclient kubernetes.Interface, namespace string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		sa, err := kubeclient.CoreV1().ServiceAccounts(namespace).Get(toolchainenabler.SAName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s Service Account in namespace %s \n", toolchainenabler.SAName, namespace)
				return false, nil
			}
			return false, err
		}

		if sa != nil {
			t.Logf("Found Service Account %s in namespace %s \n", toolchainenabler.SAName, namespace)
			return true, nil
		}

		t.Logf("Waiting for Service Account %s \n", toolchainenabler.SAName)
		return false, nil
	})

	if err != nil {
		return err
	}

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		crb, err := kubeclient.RbacV1().ClusterRoleBindings().Get(toolchainenabler.CRBName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s ClusterRoleBinding\n", toolchainenabler.CRBName)
				return false, nil
			}
			return false, err
		}

		if crb != nil {
			t.Logf("Found ClusterRoleBinding %s \n", toolchainenabler.CRBName)
			return true, nil
		}

		t.Logf("Waiting for ClusterRoleBinding %s \n", toolchainenabler.CRBName)
		return false, nil
	})

	if err != nil {
		return err
	}

	t.Logf("Service Account %s is available with self-provision role \n", toolchainenabler.SAName)
	return nil
}

func waitForOauthClient(t *testing.T, framewokclient framework.FrameworkClient, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		oc := &oauthv1.OAuthClient{}
		if err := framewokclient.Get(context.Background(), types.NamespacedName{Name: toolchainenabler.OAuthClientName}, oc); err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s OAuthClient \n", toolchainenabler.OAuthClientName)
				return false, nil
			}
			return false, err
		}

		if !reflect.DeepEqual(oauthv1.OAuthClient{}, *oc) {
			t.Logf("Found OAuthClient %s \n", toolchainenabler.OAuthClientName)
			return true, nil
		}
		t.Logf("Waiting for availability of %s OAuthClient \n", toolchainenabler.OAuthClientName)
		return false, nil
	})

	if err != nil {
		return err
	}

	t.Logf("OAuthClient %s is created and ready to use \n", toolchainenabler.OAuthClientName)
	return nil
}
