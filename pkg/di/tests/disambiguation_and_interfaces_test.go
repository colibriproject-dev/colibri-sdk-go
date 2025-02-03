package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_Interfaces_Disambiguation_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newFooImplementation1)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Tag_not_found(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newFooImplementation1, newFooImplementation3)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithTag) })
}

func Test_Interfaces_Disambiguation_Not_Tag(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newFooImplementation1, newFooImplementation2)
	assert.Panics(t, func() { a.StartApp(NewBarObjectWithoutTag) })
}

func Test_Interfaces_Disambiguation_Sucess_2(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newFooImplementation1, newFooImplementation2)
	assert.NotPanics(t, func() { a.StartApp(NewBarObjectWithTag) })
}
