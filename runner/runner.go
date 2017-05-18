package main

import (
	"fmt"

	"github.com/pcrawfor/fayego/server"
)

func main() {
	port := ":8000"
	fmt.Println("Starting faye server on port", port)
	server.Start(port)
}
