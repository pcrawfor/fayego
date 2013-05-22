package main

import (
	"encoding/json"
	"fmt"
	"github.com/pcrawfor/fayego/fayeclient"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Faye Client Runner start")

	client, err := fayeclient.NewFayeClient()
	if err != nil {
		fmt.Println("Error creating client")
		os.Exit(0)
	}

	msgData := map[string]string{
		"message": "hi",
	}

	j, _ := json.Marshal(msgData)
	client.Write(string(j))
	client.Handshake()

	// handle interrupts
	hupChan := make(chan os.Signal, 1)
	termChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-termChan:
			fmt.Println("INT Signal")
			os.Exit(0)
		case <-hupChan:
			fmt.Println("HUP Signal")
			os.Exit(0)
		}
	}
}
