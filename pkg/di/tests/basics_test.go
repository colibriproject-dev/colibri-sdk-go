package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{beanInt}
	a.AddDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(InitializeAPP) })
}

func Test_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{beanInt, beanFloat32}
	a.AddDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(InitializeAPP) })
}
