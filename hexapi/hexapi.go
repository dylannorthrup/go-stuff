package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	//	"os"
)

// Registering these as copies that we can add ServeHTTP methods to
type String string
type Display string

// These are what we work in
type Card struct {
	Name    string
	Quantiy int
	Rarity  string
	Gold    int
	Plat    int
}

type test_struct struct {
	Test string
}

// And this is all our cards together
type Collection []Card

// FUNCTIONS

//func test(rw http.ResponseWriter, req *http.Request) {
//	decoder := json.NewDecoder(req.Body)
//	var t test_struct
//	err := decoder.Decode(&t)
//	if err != nil {
//		panic("AIEEE")
//	}
//	log.Println(t.Test)
//}

func test(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("AIEEE")
	}
	log.Println(string(body))
	var t test_struct
	err = json.Unmarshal(body, &t)
	if err != nil {
		panic("AIEEE")
	}
	log.Println(t.Test)
}

/* func response(w http.ResponseWriter, r *http.Request) {
	//	b := make([]byte, 4096)
	//	n, err := r.Body.Read(b)
	//	if err != nil {
	//		fmt.Printf("Request body: %q\n", b[:n])
	//	} else {
	//		fmt.Print("Had a problem with r.Body.Read: %v\n", err)
	//	}
	w.Write([]byte("Responding\n"))
	//fmt.Fprintf(os.Stdout, "Request body: %v\n", r.Body)
} */

// Now, let's implement 'ServeHTTP()' for them
func (s String) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "STRING: %v\n", s)
}

func main() {
	// your http.Handle calls here
	// Register http handlers before starting the server
	//	http.HandleFunc("/", response)
	http.HandleFunc("/", test)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}
