package cluster

import (
	"fmt"
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"k8s.io/api/core/v1"
)

type configOption func(data *clusterclient.CreateClusterData) error

func name(i configInformer) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		c.Name = i.clusterName
		return nil
	}
}

func apiURL(i configInformer) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		c.APIURL = fmt.Sprintf("https://api.%s.openshift.com/", i.clusterName)
		return nil
	}
}

func appDNS(i configInformer, options ...RouteOption) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		subDomain, err := routingSubDomain(i, options...)
		if err != nil {
			return err
		}
		c.AppDNS = subDomain
		return nil
	}
}

func oauthClient(i configInformer) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		c.AuthClientID = config.OAuthClientName
		c.AuthClientDefaultScope = "user:full"

		oauthClient, err := i.oc.GetOAuthClient(config.OAuthClientName)
		if err != nil {
			return err
		}

		c.AuthClientSecret = oauthClient.Secret
		return nil
	}
}

func serviceAccount(i configInformer, options ...SASecretOption) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		c.ServiceAccountUsername = fmt.Sprintf("system:serviceaccount:%s:%s", i.ns, config.SAName)
		sa, err := i.oc.GetServiceAccount(i.ns, config.SAName)
		if err != nil {
			return err
		}

		//used for testing
		for _, opt := range options {
			opt(sa)
		}

		if len(sa.Secrets) == 0 {
			return errors.Errorf("couldn't find any secret reference for sa %s", sa.Name)
		}

		var saSecret *v1.Secret
		for _, s := range sa.Secrets {
			sec, err := i.oc.GetSecret(i.ns, s.Name)
			if err != nil {
				return err
			}
			// we are not interested in `kubernetes.io/dockercfg`
			if sec.Type == v1.SecretTypeServiceAccountToken {
				saSecret = sec
			}
		}
		if saSecret == nil {
			return errors.Errorf("couldn't find any secret reference for sa %s of type %s", sa.Name, v1.SecretTypeServiceAccountToken)
		}
		c.ServiceAccountToken = string(saSecret.Data["token"])

		return nil
	}
}

func typeOSD() configOption {
	return func(c *clusterclient.CreateClusterData) error {
		c.Type = "OSD"

		return nil
	}
}

func tokenProvider() configOption {
	return func(c *clusterclient.CreateClusterData) error {
		tokenProviderID := uuid.NewV4().String()
		c.TokenProviderID = &tokenProviderID

		return nil
	}
}

type SASecretOption func(sa *v1.ServiceAccount)
