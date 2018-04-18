// Entry point for serving the analyzed bug/trello data
package main

import (
	"fmt"
	"log"
	"net/http"
)

// handler gives a simple response to all incoming requests - for testing
func handler(writ http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(writ, "Hello, world")
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
