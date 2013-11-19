package fayeclient

import (
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"io/ioutil"
	"time"
)

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

// interface responsible for parsing faye messages
type FayeHandler interface {
	HandleMessage(message []byte) error
}

type Connection struct {
	ws              *websocket.Conn
	readerConnected bool
	writerConnected bool
	send            chan string
	exit            chan bool
}

func (c *Connection) Connected() bool {
	return c.readerConnected && c.writerConnected
}

/*
reader

Read messages from the websocket connection
*/
func (c *Connection) reader(f FayeHandler) {
	fmt.Println("reading...")
	c.readerConnected = true
	defer func() {
		fmt.Println("reader disconnect")
		c.ws.Close()
		c.readerConnected = false
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

			//fmt.Println("Read message: " + string(message))
			f.HandleMessage(message)

		}
	}
	fmt.Println("reader exited.")
}

/*
  Writer

  Write messages to the websocket connection
*/
func (c *Connection) write(opCode int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(opCode, payload)
}

func (c *Connection) writer() {
	fmt.Println("Writer started.")
	ticker := time.NewTicker(pingPeriod)
	c.writerConnected = true
	defer func() {
		ticker.Stop()
		c.ws.Close()
		c.writerConnected = false
	}()
	for {
		select {
		case message, ok := <-c.send:
			//fmt.Println("writing: ", message)
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
		case <-c.exit:
			fmt.Println("exiting writer...")
			return
		}

	}
}
