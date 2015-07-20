// Methods can be associated with a named type or a pointer to a named type
// There are two reasons to use a pointer receiver. First, to avoid copying the value on each
// method call (which is more efficient if the value type is a large struct). Second, so the method
// can modify the value that its receiver points to

package main

import (
	"fmt"
	"math"
)

// Declare the Vertext struct
type Vertex struct {
	X, Y float64
}

// Make a Scale method for Vertex pointers
// This would not be possible if we weren't using pointers
func (v *Vertex) Scale(f float64) {
	v.X = v.X * f
	v.Y = v.Y * f
}

// Make an Abs method for Vertex pointers
// This would work just as well without a pointer since it doesn't modify
// any values of the method receiver
func (v *Vertex) Abs() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func main() {
	// Make a Vertex, then assign v as a pointer to that Vertex
	v := &Vertex{3, 4}
	// Use Scale to make the Vertex bigger
	v.Scale(5)
	// Print out the new, bigger Vertex and the Abs value for it
	fmt.Println(v, v.Abs())
}
