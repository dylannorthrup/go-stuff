package main

import (
	"encoding/json"
	"fmt"
	//"io"
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

// And this is all our cards together
type Collection []Card

// FUNCTIONS

func incoming(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("AIEEE: Could not readAll for req.Body")
	}
	fmt.Println(string(body))
	//err = json.Unmarshal(body, &t)
	var f map[string]interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		panic("AIEEE: Could not Unmarshall the body")
	}
	fmt.Println("PROGRESS: Unmarshall successful")
	for k, v := range f {
		fmt.Printf("working on key %v with value %v\n", k, v)
		switch vv := v.(type) {
		case string:
			fmt.Println(k, "is string", vv)
		case int:
			fmt.Println(k, "is int", vv)
		case []interface{}:
			fmt.Println(k, "is an array:")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}

}

func main() {
	// Register http handlers before starting the server
	http.HandleFunc("/", incoming)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}
