package main

import (
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func Test_metadata_Bean_not_found(t *testing.T) {
	a := di.NewContainer()
	assert.Panics(t, func() { a.StartApp(NewBeanWithMetadata) })
}

func Test_metadata_Success(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(NewBeanDependency1)
	assert.NotPanics(t, func() { a.StartApp(NewBeanWithMetadata) })
}

func Test_Success_Disambiguation(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(NewBeanDependency1, NewBeanDependency2)
	assert.NotPanics(t, func() { a.StartApp(NewBeanWithMetadata) })
}

func Test_Disambiguation_Tag_not_found(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(NewBeanDependency1, NewBeanDependency3)
	assert.Panics(t, func() { a.StartApp(NewBeanWithMetadata) })
}

func Test_Disambiguation_Not_Tag(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(NewBeanDependency1, NewBeanDependency2, NewBeanDependency3)
	assert.Panics(t, func() { a.StartApp(NewBeanWithoutMetadata) })
}
