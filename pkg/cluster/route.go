package cluster

import (
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

const RouteName = "toolchain-route"

// routingSubDomain returns default routing sub-domain configured in openshift master. For more info check https://bit.ly/2Dj2kfh
func routingSubDomain(i configInformer, options ...RouteOption) (string, error) {
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

type RouteOption func(r *routev1.Route)
