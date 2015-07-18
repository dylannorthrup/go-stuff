package main

import (
  "fmt"
  "math"
)

func absolute(x float64) float64 {
  if x < 0 {
    x = -x
  }
  return x
}

func Sqrt(x float64) float64 {
  guess := float64(1)

  for absolute(guess*guess - x) >= 0.000001{
        guess = ((x/guess) + guess) / 2;
  }

    return guess

}

func main() {
  fmt.Println(Sqrt(2), math.Sqrt(2))
  fmt.Println(Sqrt(3), math.Sqrt(3))
  fmt.Println(Sqrt(4), math.Sqrt(4))
  fmt.Println(Sqrt(5), math.Sqrt(5))
  fmt.Println(Sqrt(6), math.Sqrt(6))
  fmt.Println(Sqrt(7), math.Sqrt(7))
  fmt.Println(Sqrt(8), math.Sqrt(8))
  fmt.Println(Sqrt(9), math.Sqrt(9))
}

