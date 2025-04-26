package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Interfaces_Disambiguation_Global_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Global_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newFooImplementation1}
	a.AddGlobalDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Global_Tag_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newFooImplementation1, newFooImplementation3}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithTag) })
}

func Test_Interfaces_Disambiguation_Global_Not_Tag(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newFooImplementation1, newFooImplementation2}
	a.AddGlobalDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Global_Sucess_2(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []any{newFooImplementation1, newFooImplementation2}
	a.AddGlobalDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithTag) })
}
