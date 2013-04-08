/*
Created by Paul Crawford
Copyright (c) 2013. All rights reserved.
*/
package fayeserver

import (
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// =====
// WebSocket handling

type Connection struct {
	ws   *websocket.Conn
	send chan string
}

/* 
Initial constants based on websocket example code from github.com/garyburd/go-websocket
Reader & Writer functions also implemented based on 
*/
const (
	// Time allowed to write a message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next message from the client.
	readWait = 60 * time.Second

	// Send pings to client with this period. Must be less than readWait.
	pingPeriod = (readWait * 9) / 10

	// Maximum message size allowed from client.
	maxMessageSize = 512
)

/*
reader

Read messages from the websocket connection
*/
func (c *Connection) reader(f *FayeServer) {
	fmt.Println("reading...")
	defer func() {
		fmt.Println("reader disconnect")
		f.DisconnectChannel(c.send)
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(readWait))
	for {
		op, r, err := c.ws.NextReader()
		if err != nil {
			break
		}
		switch op {
		case websocket.OpPong:
			c.ws.SetReadDeadline(time.Now().Add(readWait))
		case websocket.OpText:
			message, err := ioutil.ReadAll(r)
			if err != nil {
				break
			}

			fmt.Println("Message: " + string(message))

			// ask faye client to handle faye message		
			response, error := f.HandleMessage(message, c.send)
			if error != nil {
				fmt.Println("Faye Error: ", error)
				c.send <- fmt.Sprintf("Error: ", error)
			} else {
				c.send <- string(response)
			}
		}
	}
}

/*
writer

Write messages to the websocket connection
*/

func (c *Connection) write(opCode int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(opCode, payload)
}

func (c *Connection) writer(f *FayeServer) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.OpClose, []byte{})
				return
			}
			if err := c.write(websocket.OpText, []byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.OpPing, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	ws, err := websocket.Upgrade(w, r.Header, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}

	c := &Connection{send: make(chan string, 256), ws: ws}
	go c.writer(f)
	c.reader(f)
}

var f *FayeServer

/*
Start the Faye server on the address/port given in the addr param
*/
func Start(addr string) {
	f = NewFayeServer()
	http.HandleFunc("/faye", serveWs)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
