package main

import "fmt"

// Define a Person struct
type Person struct {
	Name string
	Age  int
}

// This is a "Stringer".  It lets a thing print out reasonable information about itself
// This lets you pass a Person to a function expecting a String function be available
// (like fmt.Println())
func (p Person) String() string {
	return fmt.Sprintf("%v (%v years)", p.Name, p.Age)
}

func main() {
	a := Person{"Arthur Dent", 42}
	z := Person{"Zaphod Beeblebrox", 9001}
	fmt.Println(a, z)
}
