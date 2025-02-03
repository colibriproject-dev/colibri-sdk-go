package main

import (
	"fmt"
	"testing"

	"github.com/colibri-project-io/colibri-sdk-go/pkg/di"
	"github.com/stretchr/testify/assert"
)

func newInt2() int {
	return 2
}

func newInt1() int {
	return 1
}

func newStringV() string {
	return "stringv"
}

func NewV(a string, s ...int) string {
	fmt.Println("recebi: ", len(s), " dependencias ")
	return "s"
}

func Test_variadic(t *testing.T) {
	a := di.NewContainer()
	a.AddDependencies(newInt1, newInt2, newStringV)
	assert.NotPanics(t, func() { a.StartApp(NewV) })
}
