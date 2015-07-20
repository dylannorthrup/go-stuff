package main

import (
	"io"
	"os"
	"strings"
)

type rot13Reader struct {
	r io.Reader
}

func (rot rot13Reader) Read(b []byte) (n int, err error) {
	// First thing we do is use a standard io.Reader (handily provided by the struct def up there)
	// to get the raw contents of whatever we're reading and stuffing it into 'b'.
	n, err = rot.r.Read(b)

	// Once we have the raw bits inside 'b', we do some processing.
	// For each letter, and translate it up or down as appropriate. We compare the value of the byte
	// to known values to see if it's between a-m/A-M or n-z/N-Z. We don't mess with anything else
	for i := 0; i < len(b); i++ {
		if (b[i] >= 'A' && b[i] < 'N') || (b[i] >= 'a' && b[i] < 'n') {
			b[i] += 13
		} else if (b[i] > 'M' && b[i] <= 'Z') || (b[i] > 'm' && b[i] <= 'z') {
			b[i] -= 13
		}
	}
	// Could simply put 'return' here, but I like to be explicit
	return n, err
}

func main() {
	s := strings.NewReader("Lbh penpxrq gur pbqr!\n")
	r := rot13Reader{s}
	io.Copy(os.Stdout, &r)
}
