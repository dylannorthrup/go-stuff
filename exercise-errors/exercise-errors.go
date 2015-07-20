package main

import (
	"fmt"
)

type ErrNegativeSqrt float64

func Sqrt(x float64) (float64, error) {
	// Check if x is negative
	if x < 0 {
		return x, ErrNegativeSqrt(x)
	}
	guess := float64(1)

	for absolute(guess*guess-x) >= 0.000001 {
		guess = ((x / guess) + guess) / 2
	}

	return guess, nil

}

func absolute(x float64) float64 {
	if x < 0 {
		x = -x
	}
	return x
}

func (e ErrNegativeSqrt) Error() string {
	return fmt.Sprintf("cannot Sqrt negative number: %v", float64(e))
}

func main() {
	fmt.Println(Sqrt(25))
	fmt.Println(Sqrt(-2))
}
