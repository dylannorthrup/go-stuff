package main

import (
	"fmt"
	"log"
	"net/http"
)

// Registering these as copies that we can add ServeHTTP methods to
type String string

type Struct struct {
	Greeting string
	Punct    string
	Who      string
}

// Now, let's implement 'ServeHTTP()' for them
func (s String) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "STRING: %v\n", s)
}
func (s Struct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "STRUCT: %v %v %v\n", s.Greeting, s.Punct, s.Who)
}

func main() {
	// your http.Handle calls here
	// Register http handlers before starting the server
	http.Handle("/string", String("I'm a frayed knot."))
	http.Handle("/struct", &Struct{"Hello", ":", "Gophers!"}) // Not sure why I'm using the struct reference
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe("localhost:4000", nil))
}
