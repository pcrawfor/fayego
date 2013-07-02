package fayeclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"github.com/pcrawfor/fayego/fayeserver"
	"net"
	"net/url"
	"os"
)

const DEFAULT_HOST = "localhost:4001/faye"

const ( // iota is reset to 0
	StateWSDisconnected   = iota // == 0
	StateWSConnected      = iota
	StateFayeDisconnected = iota
	StateFayeConnected    = iota
)

type FayeClient struct {
	Host          string
	MessageChan   chan string // any messages recv'd by the client will be sent to the message channel - TODO: remap this to a set of subscription message channels one per active subscription
	conn          *Connection
	fayeState     int
	readyChan     chan bool
	clientId      string
	messageNumber int
}

func NewFayeClient(host string) *FayeClient {
	if len(host) == 0 {
		host = DEFAULT_HOST
	}
	// instantiate a FayeClient and return
	return &FayeClient{Host: host, fayeState: StateWSDisconnected, MessageChan: make(chan string, 100), messageNumber: 0}
}

func (f *FayeClient) Start(ready chan bool) error {
	fmt.Println("Starting...")
	err := f.connectToServer()
	if err != nil {
		return err
	}

	// kick off the connection handshake
	f.readyChan = ready
	f.handshake()
	return nil
}

func (f *FayeClient) connectToServer() error {
	fmt.Println("start client")

	url, _ := url.Parse("ws://" + f.Host)
	c, err := net.Dial("tcp", url.Host)

	if err != nil {
		fmt.Println("Error connecting to server: ", err)
		os.Exit(0)
		//panic("Dial: %v", err)
	}

	ws, resp, err := websocket.NewClient(c, url, nil, 1024, 1024)

	if err != nil {
		return err
	}

	f.fayeState = StateWSConnected

	if resp != nil {
		fmt.Println("Resp: ", resp)
	}

	conn := &Connection{send: make(chan string, 256), ws: ws, exit: make(chan bool)}
	f.conn = conn
	go conn.writer()
	go conn.reader(f)

	return nil
}

func (f *FayeClient) disconnectFromServer() {
	f.conn.exit <- true
	f.fayeState = StateWSDisconnected
	f.conn.ws.Close()
}

func (f *FayeClient) Write(msg string) error {
	//fmt.Println("Send msg: ", msg)
	f.conn.send <- msg
	return nil
}

/*
Parse and interpret a faye message response
*/
func (f *FayeClient) HandleMessage(message []byte) error {
	//fmt.Println("Handle message: ", string(message))

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

	//fmt.Println("parsed: ", resp[0].Channel)

	switch fm.Channel {
	case fayeserver.CHANNEL_HANDSHAKE:
		//fmt.Println("Recv'd handshake response")
		//fmt.Println("connect client id: ", fm.ClientId)
		f.clientId = fm.ClientId
		f.connect() // send faye connect message
		f.fayeState = StateFayeConnected
		f.readyChan <- true

	case fayeserver.CHANNEL_CONNECT:
		//fmt.Println("Recv'd connect response")
	case fayeserver.CHANNEL_DISCONNECT:
		//fmt.Println("Recv'd disconnect response")
		f.fayeState = StateFayeDisconnected
		f.disconnectFromServer()

	case fayeserver.CHANNEL_SUBSCRIBE:
		//fmt.Println("Recv'd subscribe response")
		// TODO: store the subscription state if successful

	case fayeserver.CHANNEL_UNSUBSCRIBE:
		//fmt.Println("Recv'd unsubscribe response")
		// TODO: clear the subscription state if successful
	default:
		//fmt.Println("Recv'd message on channel: ", fm.Channel)
		//fmt.Println("data is: ", fm.Data)
		if fm.Data != nil {
			if fm.ClientId == f.clientId {
				return nil
			}
			data := fm.Data.(map[string]interface{})
			m := data["message"].(string)
			// tell the client we got a message on a channel.
			// TODO: do this via the subscription management and state
			go func(msg string) {
				f.MessageChan <- msg
			}(m)
		}
	}

	return nil
}

func (f *FayeClient) Subscribe(channel string) error {
	if len(channel) == 0 {
		return errors.New("Channel must have a value.")
	}
	//fmt.Println("Subscribe to channel: ", channel)
	return f.subscribe(channel)
}

func (f *FayeClient) Unsubscribe(channel string) error {
	if len(channel) == 0 {
		return errors.New("Channel must have a value.")
	}
	//fmt.Println("Unsubscribe from channel: ", channel)
	return f.unsubscribe(channel)
}

func (f *FayeClient) Publish(channel, message string) error {
	// TODO: tell faye to publish a message on a channel.
	f.publish(channel, message)
	return nil
}

func (f *FayeClient) Disconnect() {
	f.disconnect()
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

// Faye message functions

/*
 */
func (f *FayeClient) handshake() {
	message := fayeserver.FayeResponse{Channel: fayeserver.CHANNEL_HANDSHAKE, Version: "1.0", SupportedConnectionTypes: []string{"websocket"}}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating handshake message")
	}
}

/*
Connect to Faye
*/
func (f *FayeClient) connect() {
	message := fayeserver.FayeResponse{Channel: fayeserver.CHANNEL_CONNECT, ClientId: f.clientId, ConnectionType: "websocket"}
	//fmt.Println("Connect message: ", message)
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating connect message")
	}
}

/*
Disconnect from Faye
*/
func (f *FayeClient) disconnect() {
	message := fayeserver.FayeResponse{Channel: fayeserver.CHANNEL_DISCONNECT, ClientId: f.clientId}
	//fmt.Println("Connect message: ", message)
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating connect message")
	}
}

/*
Subscribe the client to a channel
*/
func (f *FayeClient) subscribe(channel string) error {
	message := fayeserver.FayeResponse{Channel: fayeserver.CHANNEL_SUBSCRIBE, ClientId: f.clientId, Subscription: channel}
	//fmt.Println("Subscribe message: ", message)
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating subscribe message")
		return err
	}
	return nil
}

/*
Unsubscribe from a channel.
*/
func (f *FayeClient) unsubscribe(channel string) error {
	message := fayeserver.FayeResponse{Channel: fayeserver.CHANNEL_UNSUBSCRIBE, ClientId: f.clientId, Subscription: channel}
	//fmt.Println("Unsubscribe message: ", message)
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating unsubscribe message")
		return err
	}
	return nil
}

/*
  Publish a message to a channel.
*/
func (f *FayeClient) publish(channel, msg string) error {
	data := map[string]interface{}{"message": msg}
	message := fayeserver.FayeResponse{Channel: channel, ClientId: f.clientId, Id: f.messageId(), Data: data}
	//fmt.Println("publish message: ", message)
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating unsubscribe message")
		return err
	}
	return nil
}

// ------------------

/*
Encode the json and send the message over the wire.
*/
func (f *FayeClient) writeMessage(message fayeserver.FayeResponse) error {
	json, err := json.Marshal(message)
	if err != nil {
		return err
	}
	f.Write(string(json))
	return nil
}

// Message Id
func (f *FayeClient) messageId() string {
	return "1"
}

/*

TODO:

* handle connection state for FayeClient
* implement other protocol comm functions
* implement publish and subscribe semantics

*/
