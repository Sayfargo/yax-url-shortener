package main

import (
	"os"
)

// Тело функции foo() анализироваться не должно
func foo() {
	os.Exit(1)
}

type user struct{}

func (user) main() {
	os.Exit(1)
}

func main() {

	defer os.Exit(1) // want "direct call to os.Exit is forbidden"

	os.Exit(1) // want "direct call to os.Exit is forbidden"

	go os.Exit(1) // want "direct call to os.Exit is forbidden"

	foo := func() {
		os.Exit(1) // want "direct call to os.Exit is forbidden"
	}

	foo()
}
