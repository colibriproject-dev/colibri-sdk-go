package main

import (
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_global_interfaces_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}

func Test_global_interfaces_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{newMyGlobalImplementation}
	a.AddGlobalDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}
