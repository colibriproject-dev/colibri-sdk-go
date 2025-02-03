package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(beanInt)
	assert.Panics(t, func() { a.StartApp(InitializeAPP) })
}

func Test_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(beanInt, beanFloat32)
	assert.NotPanics(t, func() { a.StartApp(InitializeAPP) })
}
