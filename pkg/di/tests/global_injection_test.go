package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Global_Injection_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{GlobalBeanString, globalBeanInt}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(GlobalInitializeAPP) })
}

func Test_Global_Injection_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{globalBeanFloat32, GlobalBeanString, globalBeanInt}
	a.AddGlobalDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(GlobalInitializeAPP) })
}
