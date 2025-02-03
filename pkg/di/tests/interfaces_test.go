package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_interfaces_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	assert.Panics(t, func() { a.StartApp(NewMyDependencyObject) })
}

func Test_interfaces_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newMyImplementation)
	assert.NotPanics(t, func() { a.StartApp(NewMyDependencyObject) })
}
