package controller

import (
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/sshkeypair"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sshkeypair.Add)
}
