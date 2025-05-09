package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_global_interfaces_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}

func Test_global_interfaces_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newMyGlobalImplementation}
	a.AddGlobalDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewMyGlobalDependencyObject) })
}
