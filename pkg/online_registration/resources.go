package online_registration

import (
	"context"
	"fmt"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	errs "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("online_registration_resource_creator")

const (
	ServiceAccountName     = "online-registration"
	Namespace              = "openshift-infra"
	ClusterRoleBindingName = "online-registration"
)

var serviceAccount = corev1.ServiceAccount{
	ObjectMeta: metav1.ObjectMeta{
		Name:      ServiceAccountName,
		Namespace: Namespace,
	},
}

var clusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: ClusterRoleBindingName,
	},
	Subjects: []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      ServiceAccountName,
			Namespace: Namespace,
		},
	},
	RoleRef: rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "online-registration",
	},
}

func EnsureServiceAccount(client client.Client, cache cache.Cache) error {
	sa := &corev1.ServiceAccount{}
	if err := cache.Get(context.Background(), types.NamespacedName{Namespace: Namespace, Name: ServiceAccountName}, sa); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating a new service account ", "namespace", Namespace, "name", ServiceAccountName)
			sa := serviceAccount
			if err := client.CreateServiceAccount(&sa); err != nil {
				return err
			}
			log.Info(fmt.Sprintf("service account %s in namespace %s created successfully", ServiceAccountName, Namespace))
			return nil
		}
		return errs.Wrapf(err, "failed to get service account %s from namespace %s", ServiceAccountName, Namespace)
	}
	log.Info(fmt.Sprintf("service account %s already exists", ServiceAccountName))
	return nil
}

func EnsureClusterRoleBinding(client client.Client) error {
	if _, err := client.GetClusterRoleBinding(ClusterRoleBindingName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("adding online-registration cluster role to", "service account", ServiceAccountName, "namespace", Namespace)
			crb := clusterRoleBinding
			if err := client.CreateClusterRoleBinding(&crb); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", ClusterRoleBindingName))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", ClusterRoleBindingName)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", ClusterRoleBindingName))
	return nil
}
