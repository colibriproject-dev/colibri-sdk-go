package main

import "fmt"

func beanFloat32() float32 {
	fmt.Println("creating pop")
	return 3.2
}

func beanInt() int {
	fmt.Println("creating bar")
	return 2
}

func InitializeAPP(a int, b float32) string {
	fmt.Println("creating baz")
	return fmt.Sprintf("%d - %f", a, b)
}
