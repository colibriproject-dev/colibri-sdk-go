package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Global_Injection_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	a.AddGlobalDependencies(GlobalBeanString, globalBeanInt)
	assert.Panics(t, func() { a.StartApp(GlobalInitializeAPP) })
}

func Test_Global_Injection_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddGlobalDependencies(globalBeanFloat32, GlobalBeanString, globalBeanInt)
	assert.NotPanics(t, func() { a.StartApp(GlobalInitializeAPP) })
}
