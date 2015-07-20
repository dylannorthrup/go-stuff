package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Add a Read([]byte) (int, error) method to MyReader.

// So, important note: b is the array we're being passed. We want to modify that since that's what
// we're being asked to fill up.  We aren't returning the contents of what we read, simply how many
// things we read and whether or not we have an error.  Ideally we'd also return an EOF in the error
// if we got to the end of whatever we're reading (I think)
func (m MyReader) Read(b []byte) (n int, err error) {
	if b == nil {
		return 0, nil
	} else {
		l := len(b)
		for i, _ := range b {
			b[i] = 'A'
		}
		return l, nil
	}
}

func main() {
	reader.Validate(MyReader{})
}
