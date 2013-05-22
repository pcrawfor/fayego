package fayeclient

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"github.com/pcrawfor/fayego/fayeserver"
	"net"
	"net/url"
	"os"
)

type FayeClient struct {
	conn *Connection
}

func NewFayeClient() (*FayeClient, error) {
	fmt.Println("start client")
	url, _ := url.Parse("ws://localhost:4001/faye")
	c, err := net.Dial("tcp", url.Host)

	if err != nil {
		fmt.Println("Error connecting to server: ", err)
		os.Exit(0)
		//panic("Dial: %v", err)
	}

	ws, resp, err := websocket.NewClient(c, url, nil, 1024, 1024)

	if err != nil {
		return nil, err
	}

	if resp != nil {
		fmt.Println("Resp: ", resp)
	}

	conn := &Connection{send: make(chan string, 256), ws: ws}
	f := &FayeClient{conn}
	go conn.writer()
	go conn.reader(f)

	// instantiate a FayeClient and return
	return f, nil
}

func (f *FayeClient) Write(msg string) {
	fmt.Println("Send msg: ", msg)
	f.conn.send <- msg
}

/*
Parse and interpret a faye message response
*/
func (f *FayeClient) HandleMessage(message []byte) error {
	fmt.Println("Handle message: ", string(message))

	// parse the faye message and interpret the logic to set client state appropriately
	resp := []fayeserver.FayeResponse{}
	err := json.Unmarshal(message, &resp)
	var fm fayeserver.FayeResponse
	if len(resp) > 0 {
		fm = resp[0]
	}

	if err != nil {
		fmt.Println("Error parsing json. ", err)
	}

	fmt.Println("parsed: ", resp[0].Channel)

	switch fm.Channel {
	case "/meta/handshake":
		fmt.Println("handshake")
		//return f.handshake()
	case "/meta/connect":
		fmt.Println("connect")
		//return f.connect(fm.ClientId)
	case "/meta/disconnect":
		fmt.Println("disconnect")
		//return f.disconnect(fm.ClientId)
	case "/meta/subscribe":
		fmt.Println("subscribe")
		//return f.subscribe(fm.ClientId, fm.Subscription, c)
	case "/meta/unsubscribe":
		fmt.Println("subscribe")
		//return f.unsubscribe(fm.ClientId, fm.Subscription)
	default:
		fmt.Println("publish")
		fmt.Println("data is: ", fm.Data)
		//return f.publish(fm.Channel, fm.Id, fm.Data)
	}

	return nil
}

/*
Faye protocol messages
*/

/*
type FayeResponse struct {
	Channel                  string            `json:"channel,omitempty"`
	Successful               bool              `json:"successful,omitempty"`
	Version                  string            `json:"version,omitempty"`
	SupportedConnectionTypes []string          `json:"supportedConnectionTypes,omitempty"`
	ClientId                 string            `json:"clientId,omitempty"`
	Advice                   map[string]string `json:"advice,omitempty"`
	Subscription             string            `json:"subscription,omitempty"`
	Error                    string            `json:"error,omitempty"`
	Id                       string            `json:"id,omitempty"`
	Data                     interface{}       `json:"data,omitempty"`
}
*/

func (f *FayeClient) Handshake() {
	message := fayeserver.FayeResponse{Channel: "/meta/handshake", Version: "1.0", SupportedConnectionTypes: []string{"websocket"}}
	fmt.Println("Handshake message: ", message)
	json, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error generating handshake message")
	}
	f.Write(string(json))
}

/*

TODO:

* handle connection state for FayeClient
* implement other protocol comm functions
* implement publish and subscribe semantics

*/
