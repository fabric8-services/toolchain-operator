package cluster

import (
	"fmt"
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"
)

type configOption func(data *clusterclient.CreateClusterData) error

func clusterNameAndAPIURL(i configInformer) configOption {
	return func(c *clusterclient.CreateClusterData) error {
		infrastructure, err := i.oc.GetInfrastructure("cluster")
		if err != nil {
			// Openshift 3 doesn't have infrastucture resource named cluster. This is workaround for our tests to run on minishift
			if errors.IsNotFound(err) && infrastructure == nil {
				apiURL := fmt.Sprintf("https://api.%s.openshift.com/", i.clusterName)
				// To Do change to warning when we moved to logrus implementation
				log.Info("forming cluster url using given cluster name for openshift 3 clusters", "cluster_name", i.clusterName, "cluster_url", apiURL)
				c.Name = i.clusterName
				c.APIURL = apiURL
				return nil
			}
			return errs.Wrapf(err, "failed to get infrastructure resource named cluster ")
		}
		if infrastructure == nil {
			return errs.New("something went wrong, couldn't get infrastructure resource named cluster")
		}
		c.APIURL = infrastructure.Status.APIServerURL
		c.Name = extractNameFromAPIURL(c.APIURL)
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
			return errs.Errorf("couldn't find any secret reference for sa %s", sa.Name)
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
			return errs.Errorf("couldn't find any secret reference for sa %s of type %s", sa.Name, v1.SecretTypeServiceAccountToken)
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

func extractNameFromAPIURL(url string) string {
	if s := strings.Split(url, "."); len(s) > 2 {
		return strings.TrimSpace(s[1])
	}

	return ""
}
