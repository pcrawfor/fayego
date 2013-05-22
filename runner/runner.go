package main

import (
	"fmt"
	"github.com/pcrawfor/fayego/fayeserver"
)

func main() {
	fmt.Println("Starting faye server on port 4001")
	fayeserver.Start(":4001")
}
