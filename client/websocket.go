package client

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

/*
Initial constants based on websocket example code from github.com/gorilla/websocket/blob/master/examples/chat/conn.go
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

// BayeuxHandler is an interface responsible for parsing bayeux messages
type BayeuxHandler interface {
	HandleMessage(message []byte) error
	ReaderDisconnect()
}

// Connection holds a websocket connnection and manages connection state
type Connection struct {
	ws              *websocket.Conn
	readerConnected bool
	writerConnected bool
	send            chan []byte
	exit            chan bool
}

// NewConnection instantiates and returns a new Connection object
func NewConnection(ws *websocket.Conn) *Connection {
	return &Connection{send: make(chan []byte, 256), ws: ws, exit: make(chan bool)}
}

// Connected returns a bool indicating the connection state of both the reader and writer on the connection
func (c *Connection) Connected() bool {
	return c.readerConnected && c.writerConnected
}

// reader - Read messages from the websocket connection
func (c *Connection) reader(f BayeuxHandler) {
	fmt.Println("reading...")
	c.readerConnected = true

	defer func() {
		fmt.Println("reader disconnect")
		c.ws.Close()
		c.readerConnected = false
		f.ReaderDisconnect()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			fmt.Println("READ ERROR: ", err)
			break
		}

		f.HandleMessage(message)
	}

	fmt.Println("reader exited.")
}

// write messages to the websocket connection
func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writer starts a write loop that writes out messages to the websocket connection
func (c *Connection) writer() {
	fmt.Println("Writer started.")
	c.writerConnected = true

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
		c.writerConnected = false
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		case <-c.exit:
			fmt.Println("exiting writer...")
			return
		}
	}
}
