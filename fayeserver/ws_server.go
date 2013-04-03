/*
Created by Paul Crawford
Copyright (c) 2013. All rights reserved.
*/
package fayeserver

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net/http"
	"os"
)

// =====
// WebSocket handling

type Connection struct {
	ws   *websocket.Conn
	send chan string
}

/*
reader

Read messages from the websocket connection
*/
func (c *Connection) reader(f *FayeServer) {
	fmt.Println("reading...")
	for {
		var message string
		err := websocket.Message.Receive(c.ws, &message)
		if err != nil {
			fmt.Println("Reader disconnect: ", err)
			// TODO: handle disconnect
			break
		}
		fmt.Println("Message: " + message)

		// ask faye client to handle faye message		
		response, error := f.HandleMessage([]byte(message), c.send)
		if error != nil {
			fmt.Println("Faye Error: ", error)
			c.send <- fmt.Sprintf("Error: ", error)
		} else {
			c.send <- string(response)
		}
	}

	c.ws.Close()
}

/*
writer

Write messages to the websocket connection
*/
func (c *Connection) writer(f *FayeServer) {
	for message := range c.send {
		err := websocket.Message.Send(c.ws, message)
		if err != nil {
			fmt.Println("Writer disconnect: ", err)
			// TODO: handle disconnect
			break
		}
	}
	c.ws.Close()
}

func wsHandler(ws *websocket.Conn) {
	c := &Connection{send: make(chan string, 256), ws: ws}
	//f.AddConnection(c)
	go c.writer(f)
	c.reader(f)
}

// =====

var f *FayeServer

func Start() {
	f = NewFayeServer()
	http.Handle("/faye", websocket.Handler(wsHandler))
	err := http.ListenAndServe(":4001", nil)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
