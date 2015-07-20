package main

import (
	"fmt"
	"math"
)

// Assign function to a variable, then use that variable to execute that function
func main() {
	hypot := func(x, y float64) float64 {
		return math.Sqrt(x*x + y*y)
	}

	fmt.Println(hypot(3, 4))
}
