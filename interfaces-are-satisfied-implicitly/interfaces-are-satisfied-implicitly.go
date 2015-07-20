package main

import (
	"fmt"
	"os"
)

// Define Reader interface
type Reader interface {
	Read(b []byte) (n int, err error)
}

// Define Writer interface
type Writer interface {
	Write(b []byte) (n int, err error)
}

// Define ReadWriter interface as a combination of Reader and Writer
type ReadWriter interface {
	Reader
	Writer
}

func main() {
	var w Writer

	// There's no need to explicitly tag os.Stdout as a Writer interface
	// os.Stdout implements Writer because it implements Write(b []byte) (n int, err error)
	w = os.Stdout

	fmt.Fprintf(w, "hello, writer\n")
}
