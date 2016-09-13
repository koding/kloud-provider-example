// Package example
package example

import (
	"koding/kites/kloud/provider"
	"koding/kites/kloud/stack"
)

func init() {
	// Register "example" provider.
	//
	//   example_ resources
	//   jCredential.provider = example
	//   TODO
	//
	provider.All["example"] = func(bp *provider.BaseProvider) stack.Provider {
		return &Provider{
			BaseProvider: bp,
		}
	}
}
