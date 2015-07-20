package main

import (
	//	"fmt"
	"golang.org/x/tour/wc"
	"strings"
)

// Implement a word count function that returns a map of the individual word occurences in the string
func WordCount(s string) map[string]int {
	m := make(map[string]int)
	// Split the string we're given into an array
	words := strings.Fields(s)
	for _, word := range words {
		v, ok := m[word]
		//		fmt.Println("Checking ok is", ok, "for", word)
		if ok {
			//			fmt.Println("ok should be true. Adding 1 to", v)
			m[word] = v + 1
		} else {
			//			fmt.Println("ok should be false. Setting v to 1")
			m[word] = 1
		}
	}
	return m
}

func main() {
	wc.Test(WordCount)
}
