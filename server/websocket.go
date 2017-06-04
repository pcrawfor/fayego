package server

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

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
	maxMessageSize = 1024
)

/*
reader - reads messages from the websocket connection and passes them through to the server message handler
*/
func (c *Connection) reader(f *Server) {
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
		mtype, message, err := c.ws.ReadMessage()
		fmt.Println("MTYPE:", mtype)
		fmt.Println("MESSAGE:", message)
		fmt.Println("Message size:", len(string(message)))
		fmt.Println("ERR:", err)
		if err != nil {
			break
		}

		// ask faye server to handle faye message
		response, ferr := f.HandleMessage(message, c.send)
		if ferr != nil {
			fmt.Println("Faye Error: ", ferr)
			c.send <- []byte(fmt.Sprint("Error: ", ferr))
		} else {
			fmt.Println("RESPONSE:", string(response))
			c.send <- response
		}
	}

	fmt.Println("reader exited.")
}

/*
write - writes messages to the websocket connection
*/
func (c *Connection) wsWrite(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

/*
writer - is the write loop that reads messages off the send channel and writes them out over the websocket connection
*/
func (c *Connection) writer(f *Server) {
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
				c.wsWrite(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.wsWrite(websocket.TextMessage, []byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.wsWrite(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
