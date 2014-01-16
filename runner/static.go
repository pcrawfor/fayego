package main

import "net/http"

func main() {
	panic(http.ListenAndServe(":8080", http.FileServer(http.Dir("/Users/paul/go/src/github.com/pcrawfor/fayego/runner"))))
}
