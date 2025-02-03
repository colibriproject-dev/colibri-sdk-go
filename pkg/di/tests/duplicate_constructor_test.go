package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Duplicate_constructor(t *testing.T) {
	a := di.NewContainer()
	assert.Panics(t, func() {
		a.AddDependencies(beanInt, beanFloat32)
		a.AddGlobalDependencies(beanInt, beanFloat32)
		a.StartApp(InitializeAPP)
	})
}
