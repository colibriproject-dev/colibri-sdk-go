package main

import (
	"fmt"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
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

// receivedVariadic captures the values injected into NewV so tests can assert the
// resolution order. Each test resets it before running the container.
var receivedVariadic []int

func NewV(a string, s ...int) string {
	receivedVariadic = s
	fmt.Println("received: ", len(s), " dependencies ")
	return "s"
}

func Test_variadic(t *testing.T) {
	a := di.NewContainer()
	// Creating an array of functions of different types
	funcs := []any{newInt1, newInt2, newStringV}
	a.AddDependencies(funcs)
	receivedVariadic = nil
	assert.NotPanics(t, func() { a.StartApp(NewV) })
	// Registration order is newInt1 (1) then newInt2 (2); resolution must follow it.
	assert.Equal(t, []int{1, 2}, receivedVariadic)
}

func Test_variadic_order_is_deterministic(t *testing.T) {
	for i := range 50 {
		a := di.NewContainer()
		funcs := []any{newInt1, newInt2, newStringV}
		a.AddDependencies(funcs)
		receivedVariadic = nil
		a.StartApp(NewV)
		assert.Equal(t, []int{1, 2}, receivedVariadic, "iteration %d produced a different order", i)
	}
}
