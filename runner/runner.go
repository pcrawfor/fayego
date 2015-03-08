package main

import (
	"fmt"

	"github.com/pcrawfor/fayego/fayeserver"
)

func main() {
	port := ":8000"
	fmt.Println("Starting faye server on port", port)
	fayeserver.Start(port)
}
