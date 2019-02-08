package cluster

import (
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("cluster_config_informer")

const RouteName = "toolchain-route"

type Informer struct {
	oc client.Client
	ns string
}

type RouteOption func(r *routev1.Route)

func NewInformer(oc client.Client, ns string) Informer {
	return Informer{oc, ns}
}

// routingSubDomain returns default routing subdomain configured in openshift master. For more info check https://bit.ly/2Dj2kfh
func (i Informer) routingSubDomain(options ...RouteOption) (string, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RouteName,
			Namespace: i.ns,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "toolchain",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("https"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}

	// used for testing
	for _, opt := range options {
		opt(route)
	}

	if err := i.oc.CreateRoute(route); err != nil {
		return "", err
	}

	defer func() {
		if err := i.oc.DeleteRoute(route); err != nil {
			log.Error(err, "failed to delete route", "RouteName", RouteName)
		}
	}()

	return routeHostSubDomain(route.Spec.Host), nil
}

func routeHostSubDomain(h string) string {
	if s := strings.SplitAfterN(h, ".", 2); len(s) == 2 {
		return strings.TrimSpace(s[1])
	}

	return ""
}
