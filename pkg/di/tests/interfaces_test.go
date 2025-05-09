package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_interfaces_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{}
	a.AddDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewMyDependencyObject) })
}

func Test_interfaces_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newMyImplementation}
	a.AddDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewMyDependencyObject) })
}
