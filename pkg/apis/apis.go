package apis

import (
	oauthv1 "github.com/openshift/api/oauth/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	// add OAuthClient schema as it's openshift specific resource
	AddToSchemes.Register(oauthv1.Install)
	return AddToSchemes.AddToScheme(s)
}
