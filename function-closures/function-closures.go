package main

import "fmt"

// An example of a closure. If adder is assigned to a variable, 'sum' is persistent
// across invocations of the function
func adder() func(int) int {
	sum := 0
	return func(x int) int {
		sum += x
		return sum
	}
}

func main() {
	// Assign two instances of adder to two different variables
	pos, neg := adder(), adder()
	// Iterate through numbers and use those numbers with the two different closures
	for i := 0; i < 10; i++ {
		fmt.Println(
			pos(i),
			neg(-2*i),
		)
	}
}
