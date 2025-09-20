package main

import "fmt"

// Definition of an interface
type MyInterface interface {
	MyMethod() string
}

// Creating a struct that depends on this interface
type MyDependencyObject struct {
	M MyInterface
}

// Creating the constructor of this dependent struct
func NewMyDependencyObject(m MyInterface) MyDependencyObject {
	fmt.Println("creating MyDependencyObject and injecting dependencies")
	return MyDependencyObject{M: m}
}

// Definition of a struct that implements the interface
type MyImplementation struct{}

func (m MyImplementation) MyMethod() string {
	fmt.Println("creating MyImplementation")
	return "MyImplementation implementing MyMethod"
}

// Creating a constructor for Mystruct
func newMyImplementation() MyImplementation {
	return MyImplementation{}
}
