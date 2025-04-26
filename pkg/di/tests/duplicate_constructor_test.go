package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Duplicate_constructor(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{beanInt, beanFloat32}
	assert.Panics(t, func() {
		a.AddDependencies(funcs)
		a.AddGlobalDependencies(funcs)
		a.StartApp(InitializeAPP)
	})
}
