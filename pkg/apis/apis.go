package apis

import (
	configv1 "github.com/openshift/api/config/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	// register openshift specific resource
	AddToSchemes.Register(oauthv1.Install)
	AddToSchemes.Register(configv1.Install)
	AddToSchemes.Register(routev1.Install)
	return AddToSchemes.AddToScheme(s)
}
