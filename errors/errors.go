package main

import (
	"fmt"
	"time"
)

// New struct to represent an error
type MyError struct {
	When time.Time
	What string
}

// Stringer for MyError
func (e *MyError) Error() string {
	return fmt.Sprintf("at %v, %s",
		e.When, e.What)
}

// Like the Stringer interface, fmt looks for errors as well
// This is an example of a function that returns an error
func run() error {
	return &MyError{
		time.Now(),
		"it didn't work",
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}
