/*
Demonstrates some sample use of the fayego client.

Simple command line client for faye using the fayego client:
- Connects to a faye server at localhost:5222/faye
- Subscribes to a /testing channel
- Allows you to view any message sent on the channel and send your own messages to the channel

*/

package main

import (
	"bufio"
	"fmt"
	"github.com/pcrawfor/fayego/fayeclient"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Faye Client Runner start")

	client := fayeclient.NewFayeClient("localhost:5222/faye")

	ready := make(chan bool)
	err := client.Start(ready)
	if err != nil {
		fmt.Println("Error starting client: ", err)
		os.Exit(0)
	}

	// ready will recieve true when the client is connected
	<-ready
	fmt.Println("Connected to faye!")

	// subscribe to a channel
	client.Subscribe("/testing")
	client.Subscribe("/foobar")

	// read from stdin
	fmt.Print("Ready.\n> ")
	go read(client)

	go recvMessages(client)

	// handle interrupts
	hupChan := make(chan os.Signal, 1)
	termChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-termChan:
			fmt.Println("INT Signal")
			quit(client)
		case <-hupChan:
			fmt.Println("HUP Signal")
			quit(client)
		}
	}
}

/*
Listen for messages from the client's message channel and print them to stdout
*/
func recvMessages(client *fayeclient.FayeClient) {
	for {
		select {
		case message, ok := <-client.MessageChan:
			if !ok {
				fmt.Println("error on message.")
			}
			fmt.Print("\nchannel " + message.Channel + ": " + message.Data["message"].(string) + "\n> ")
		}
	}
}

func quit(client *fayeclient.FayeClient) {
	client.Unsubscribe("/testing")
	client.Disconnect()
	os.Exit(0)
}

// read from stdin
func read(client *fayeclient.FayeClient) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		m := s.Text()
		fmt.Print("me: " + m + "\n> ")
		client.Publish("/testing", m)
	}

	if err := s.Err(); err != nil {
		fmt.Println("error: ", err)
		return
	}
}
