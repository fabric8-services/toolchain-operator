package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, cache.Cache) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c cache.Cache) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, c); err != nil {
			return err
		}
	}
	return nil
}
