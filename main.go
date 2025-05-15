package main

import (
	"LucasRiboli/AtomicCircuit/pkg"
	"fmt"
)

func main() {
	fmt.Println("a")
	cb := pkg.NewCircuitBreaker(1, 2, 3)
	cb.Execute("")
}
