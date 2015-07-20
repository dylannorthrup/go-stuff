package main

import (
	"fmt"
	"golang.org/x/tour/pic"
)

// Should return a slice of length dy
// each element of the slice should be a slice of dx 8-bit unsigned integers
func Pic(dx, dy int) [][]uint8 {
	// Make slice of length dy
	sl := make([][]uint8, dy)
	count := 0
	for count < dy {
		ns := make([]uint8, dx)
		for k, v := range ns {
			v = uint8(k) * uint8(count)
			//			fmt.Println("Adding %v at index %v to %v", v, k, ns)
			//			fmt.Println("dx is %v and dy is %v", dx, dy)
			ns[k] = v
		}
		sl[count] = ns
		fmt.Println("Count is %v", count)
		count = count + 1
	}
	return sl
}

func main() {
	pic.Show(Pic)
}
