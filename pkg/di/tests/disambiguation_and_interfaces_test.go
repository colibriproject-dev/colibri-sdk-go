package main

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Interfaces_Disambiguation_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{}
	a.AddDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Success(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{newFooImplementation1}
	a.AddDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Tag_not_found(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{newFooImplementation1, newFooImplementation3}
	a.AddDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithTag) })
}

func Test_Interfaces_Disambiguation_Not_Tag(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{newFooImplementation1, newFooImplementation2}
	a.AddDependencies(funcs)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Sucess_2(t *testing.T) {
	a := di.NewContainer()
	// Criação de um array de funções de diferentes tipos
	funcs := []interface{}{newFooImplementation1, newFooImplementation2}
	a.AddDependencies(funcs)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithTag) })
}
