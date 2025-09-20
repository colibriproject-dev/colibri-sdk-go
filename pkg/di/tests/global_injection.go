package main

import "fmt"

func globalBeanFloat32() float32 {
	fmt.Println("creating globalBeanFloat32")
	return 3.2
}

func GlobalBeanString() string {
	fmt.Println("creating GlobalBeanString")
	return "value"
}

func globalBeanInt(s string) int {
	fmt.Println("creating globalBeanInt")
	return 2
}

func GlobalInitializeAPP(a int, b float32, s string) string {
	fmt.Println("creating GlobalInitializeAPP")
	return fmt.Sprintf("%d - %f", a, b)
}
