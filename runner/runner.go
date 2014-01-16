package main

import (
	"fmt"
	"github.com/pcrawfor/fayego/fayeserver"
)

func main() {
	fmt.Println("Starting faye server on port 3000")
	fayeserver.Start(":3002")
}
