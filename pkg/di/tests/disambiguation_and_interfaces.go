package main

import "fmt"

// Definition of an interface
type fooInterface interface {
	MyMethod() string
}

// Creating a struct that depends on this interface
type barObjectWithoutTag struct {
	F fooInterface
}

type barObjectWithTag struct {
	G fooInterface `di:"newFooImplementation2"`
}

// Creating the constructor of this dependent struct
func NewBarObjectWithoutTag(f fooInterface) barObjectWithoutTag {
	fmt.Println("creating barObjectWithoutTag and injecting dependencies")
	return barObjectWithoutTag{F: f}
}

// Creating the constructor of this dependent struct
func NewBarObjectWithTag(f fooInterface) barObjectWithTag {
	fmt.Println("creating barObjectWithTag and injecting dependencies")
	return barObjectWithTag{G: f}
}

// Definition of a struct that implements the interface
type fooImplementation struct{}

func (f fooImplementation) MyMethod() string {
	fmt.Println("creating fooImplementation")
	return "fooImplementation implementing MyMethod"
}

// Creating a constructor for Mystruct
func newFooImplementation1() fooImplementation {
	return fooImplementation{}
}

func newFooImplementation2() fooImplementation {
	return fooImplementation{}
}

func newFooImplementation3() fooImplementation {
	return fooImplementation{}
}
