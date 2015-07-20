package main

import (
	"fmt"
	"math"
)

// You can declare a method on any type declared in your package
// However, you can't define a method on a type from another package including built in types
// That's why we make our own type here
type MyFloat float64

// Using our new type, we make an Abs() method for it
func (f MyFloat) Abs() float64 {
	if f < 0 {
		return float64(-f)
	}
	return float64(f)
}

func main() {
	f := MyFloat(-math.Sqrt2)
	// Print out the raw version and the Absolute version
	fmt.Println(f, f.Abs())
}
