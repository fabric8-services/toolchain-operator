package test

import (
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

type Environment struct {
	k string
	v string
}

func Env(k, v string) Environment {
	return Environment{k, v}
}

func SetEnv(environments ...Environment) func() {
	originalValues := make([]Environment, 0, len(environments))

	for _, env := range environments {
		originalValues = append(originalValues, Env(env.k, os.Getenv(env.k)))
		os.Setenv(env.k, env.v)
	}
	return func() {
		for _, env := range originalValues {
			os.Setenv(env.k, env.v)
		}
	}
}

func SASecretOption(t *testing.T, cl client.Client, ns string) func(sa *corev1.ServiceAccount) {
	err := cl.CreateSecret(Secret("toolchain-sre-1fgd3", ns, "mysatoken", corev1.SecretTypeServiceAccountToken))
	require.NoError(t, err)

	err = cl.CreateSecret(Secret("toolchain-sre-6756s", ns, "mydockertoken", corev1.SecretTypeDockercfg))
	require.NoError(t, err)

	return func(sa *corev1.ServiceAccount) {
		if sa != nil {
			sa.Secrets = append(sa.Secrets,
				corev1.ObjectReference{Name: "toolchain-sre-1fgd3", Namespace: ns, Kind: "Secret"},
				corev1.ObjectReference{Name: "toolchain-sre-6756s", Namespace: ns, Kind: "Secret"},
			)
		}
	}
}

func Secret(name, namespace, token string, secretType corev1.SecretType) *corev1.Secret {
	d := make(map[string][]byte)
	d["token"] = []byte(token)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: d,
		Type: secretType,
	}
}
