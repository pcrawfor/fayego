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
	"os"
	"os/signal"
	"syscall"

	"github.com/pcrawfor/fayego/client"
)

func main() {
	fmt.Println("Faye Client Runner start")

	client := client.NewClient("localhost:8000/faye")

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
func recvMessages(client *client.Client) {
	for {
		select {
		case message, ok := <-client.MessageChan:
			if !ok {
				fmt.Println("error on message.")
			}
			fmt.Println("\nchannel ", message.Channel)
			fmt.Println("message: ", message.Data)
		}
	}
}

func quit(client *client.Client) {
	client.Unsubscribe("/testing")
	client.Disconnect()
	os.Exit(0)
}

// read from stdin
func read(client *client.Client) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		m := s.Text()
		fmt.Print("me: " + m + "\n> ")
		client.Publish("/testing", map[string]interface{}{"message": m})
	}

	if err := s.Err(); err != nil {
		fmt.Println("error: ", err)
		return
	}
}
