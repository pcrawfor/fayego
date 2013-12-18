/*
Created by Paul Crawford
Copyright (c) 2013. All rights reserved.
*/
package fayeserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"time"
)

// =====
// WebSocket handling

type Connection struct {
	ws   *websocket.Conn
	send chan []byte
}

/*
Initial constants based on websocket example code from github.com/garyburd/go-websocket
Reader & Writer functions also implemented based on
*/
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

/*
reader - reads messages from the websocket connection and passes them through to the fayeserver message handler
*/
func (c *Connection) reader(f *FayeServer) {
	fmt.Println("reading...")
	defer func() {
		fmt.Println("reader disconnect")
		f.DisconnectChannel(c.send)
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}

		// ask faye client to handle faye message
		response, ferr := f.HandleMessage(message, c.send)
		if ferr != nil {
			fmt.Println("Faye Error: ", ferr)
			c.send <- []byte(fmt.Sprintf("Error: ", ferr))
		} else {
			c.send <- response
		}
	}

	fmt.Println("reader exited.")
}

/*
write - writes messages to the websocket connection
*/
func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

/*
writer - is the write loop that reads messages off the send channel and writes them out over the websocket connection
*/
func (c *Connection) writer(f *FayeServer) {
	fmt.Println("Writer started.")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, []byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

/*
serverWs - provides an http handler for upgrading a connection to a websocket connection
*/
func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	/*
	   if r.Header.Get("Origin") != "http://"+r.Host {
	           http.Error(w, "Origin not allowed", 403)
	           return
	   }
	*/
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}
	c := &Connection{send: make(chan []byte, 256), ws: ws}
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
