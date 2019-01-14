package controller

import (
	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, toolchainenabler.Add)
}
