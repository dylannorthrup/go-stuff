package main

import "fmt"

// fibonacci is a function that returns
// a function that returns an int.
func fibonacci() func(int) int {
	return func(x int) int {
		// First two numbers of the fibonacci sequence
		num1 := 0
		num2 := 1
		// If we get either of those as an 'x', return the appropriate value
		if x == 0 {
			return num1
		}
		if x == 1 {
			return num2
		}
		// Ottherwise, start summing up
		sum := num1 + num2
		for x > 1 {
			sum = num1 + num2
			num1 = num2
			num2 = sum
			x--
		}
		return sum
	}
}

func main() {
	f := fibonacci()
	for i := 0; i < 20; i++ {
		fmt.Println(f(i))
	}
}
