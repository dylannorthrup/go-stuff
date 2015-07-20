package main

import "fmt"

func main() {
	// Make the 'pow' slice of ints
	pow := make([]int, 10)
	// Here we pump in values to the slice
	for i := range pow {
		pow[i] = 1 << uint(i)
	}
	// We're only interested in the values, so we use '_' to skip keeping track of the indices
	for _, value := range pow {
		fmt.Printf("%d\n", value)
	}
}
