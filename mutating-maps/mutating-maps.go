package main

import "fmt"

func main() {
	m := make(map[string]int)

	// Insert a value by assignment
	m["Answer"] = 42
	fmt.Println("The value:", m["Answer"])

	// Change a value by assignment as well
	m["Answer"] = 48
	fmt.Println("The value:", m["Answer"])

	// Use 'delete(map, key)' to remove a key/value from the map
	delete(m, "Answer")
	fmt.Println("The value:", m["Answer"])

	// Testing if a key is present in the map. If so, 'v' contains the value of the element
	// and 'ok' is true. If not, 'v' is the zero value for the element's type and 'ok' is false
	v, ok := m["Answer"]
	fmt.Println("The value:", v, "Present?", ok)
}
