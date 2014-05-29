package main

import (
	"fmt"
	"github.com/pcrawfor/fayego/fayeserver"
)

func main() {
	fmt.Println("Starting faye server on port 3002")
	fayeserver.Start(":3002")
}
