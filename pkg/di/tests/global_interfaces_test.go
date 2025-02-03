package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_global_interfaces_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	assert.Panics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}

func Test_global_interfaces_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddGlobalDependencies(newMyGlobalImplementation)
	assert.NotPanics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}
