package cluster

import (
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("cluster_config_informer")

type configInformer struct {
	oc          client.Client
	ns          string
	clusterName string
}

type ConfigInformer interface {
	Inform(options ...SASecretOption) (*clusterclient.CreateClusterData, error)
}

func NewConfigInformer(oc client.Client, ns string, clusterName string) ConfigInformer {
	return configInformer{oc, ns, clusterName}
}

func (i configInformer) Inform(options ...SASecretOption) (*clusterclient.CreateClusterData, error) {
	return buildClusterConfiguration(
		appDNS(i),
		clusterNameAndAPIURL(i),
		oauthClient(i),
		serviceAccount(i, options...),
		tokenProvider(),
		typeOSD(),
	)
}

func buildClusterConfiguration(opts ...configOption) (*clusterclient.CreateClusterData, error) {
	var cluster clusterclient.CreateClusterData
	for _, opt := range opts {
		err := opt(&cluster)
		if err != nil {
			return nil, err
		}
	}
	return &cluster, nil
}
