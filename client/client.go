package client

/*

TODO:

* handle extensions
* implement other protocol comm functions

*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pcrawfor/fayego/server"
	"github.com/pcrawfor/fayego/shared"
)

const defaultHost = "localhost:4001/faye"
const defaultKeepAliveSecs = 30

const ( // iota is reset to 0
	stateWSDisconnected     = iota // == 0
	stateWSConnected        = iota
	stateBayeuxDisconnected = iota
	stateBayeuxConnected    = iota
)

// Subscription is a subscription represents a subscription to a channel by the client
// Each sub has a path representing the channel on bayeux, a messageChan which is recieve any messages sent to it and a connected indicator to indicate the state of the sub on the bayeux server
type Subscription struct {
	channel   string
	connected bool
}

func (f *Client) addSubscription(channel string) {
	c := Subscription{channel: channel, connected: false}
	f.subscriptions = append(f.subscriptions, &c)
}

func (f *Client) removeSubscription(channel string) {
	for i, sub := range f.subscriptions {
		if channel == sub.channel {
			f.subscriptions = append(f.subscriptions[:i], f.subscriptions[i+1:]...)
		}
	}
}

func (f *Client) updateSubscription(channel string, connected bool) {
	s := f.getSubscription(channel)
	s.connected = connected
}

func (f *Client) getSubscription(channel string) *Subscription {
	for i, sub := range f.subscriptions {
		if channel == sub.channel {
			return f.subscriptions[i]
		}
	}
	return nil
}

func (f *Client) resubscribeSubscriptions() {
	for _, sub := range f.subscriptions {
		fmt.Println("resubscribe: ", sub.channel)
		f.subscribe(sub.channel)
	}
}

// Client manages a client connection and interactions
type Client struct {
	Host          string
	MessageChan   chan Message // any messages recv'd by the client will be sent to the message channel - TODO: remap this to a set of subscription message channels one per active subscription
	conn          *Connection
	bayeuxState   int
	readyChan     chan bool
	clientID      string
	messageNumber int
	subscriptions []*Subscription
	keepAliveSecs int
	keepAliveChan chan bool
	core          shared.Core
}

// Message represents a message recv'd via the bayeux connection
type Message struct {
	Channel string
	Data    map[string]interface{}
	Ext     map[string]interface{}
}

// NewClient returns and instance of Client
func NewClient(host string) *Client {
	if len(host) == 0 {
		host = defaultHost
	}
	// instantiate a Client and return
	return &Client{Host: host, bayeuxState: stateWSDisconnected, MessageChan: make(chan Message, 100), messageNumber: 0, keepAliveSecs: defaultKeepAliveSecs, keepAliveChan: make(chan bool), core: shared.Core{}}
}

func (f *Client) SetKeepAliveIntervalSeconds(secs int) {
	f.keepAliveSecs = secs
}

func (f *Client) Start(ready chan bool) error {
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

// connectToServer opens the websocket connection to the bayeux server and initialize the client state
func (f *Client) connectToServer() error {
	fmt.Println("start client")
	fmt.Println("connectToServer")

	url, _ := url.Parse("ws://" + f.Host)
	c, err := net.Dial("tcp", url.Host)

	if err != nil {
		fmt.Println("Error connecting to server: ", err)
		return err
	}

	ws, resp, err := websocket.NewClient(c, url, nil, 1024, 1024)

	if err != nil {
		return err
	}

	f.bayeuxState = stateWSConnected

	if resp != nil {
		fmt.Println("Resp: ", resp)
	}

	conn := NewConnection(ws)
	f.conn = conn
	f.conn.writerConnected = true
	f.conn.readerConnected = true
	go conn.writer()
	go conn.reader(f)

	// close keep alive channel to stop any running keep alive
	close(f.keepAliveChan)
	f.keepAliveChan = make(chan bool)
	go f.keepAlive()
	return nil
}

// keepalive opens and run loop on a ticker and sends keepalive messages to the server
func (f *Client) keepAlive() {
	fmt.Println("START KEEP ALIVE")
	c := time.Tick(time.Duration(f.keepAliveSecs) * time.Second)
	for {
		select {
		case _, ok := <-f.keepAliveChan:
			if !ok {
				fmt.Println("exit keep alive")
				return
			}
		case <-c:
			fmt.Println("Send keep-alive: ", time.Now())
			f.connect()
		}

	}
	fmt.Println("exiting keepalive func")
}

// disconnectFromServer closes the websocket connection and set the bayeux client state
func (f *Client) disconnectFromServer() {
	fmt.Println("DISCONNECT FROM SERVER")
	f.bayeuxState = stateWSDisconnected
	f.conn.exit <- true
	f.conn.ws.Close()
}

// ReaderDisconnect - called by the connection handler if the reader connection is dropped by the loss of a server connection
func (f *Client) ReaderDisconnect() {
	f.readyChan <- false
}

// Write sends a message to the bayeux server over the websocket connection
func (f *Client) Write(msg string) error {
	f.conn.send <- []byte(msg)
	return nil
}

// HandleMessage parses and interprets a bayeux message response
func (f *Client) HandleMessage(message []byte) error {
	// parse the bayeux message and interpret the logic to set client state appropriately
	resp := []server.BayeuxResponse{}
	err := json.Unmarshal(message, &resp)
	var fm server.BayeuxResponse

	if err != nil {
		fmt.Println("Error parsing json. ", err)
	}

	for i := range resp {
		fm = resp[i]
		switch fm.Channel {
		case shared.ChannelHandshake:
			f.clientID = fm.ClientID
			f.connect() // send bayeux connect message
			f.bayeuxState = stateBayeuxConnected
			f.readyChan <- true

		case shared.ChannelConnect:
			//fmt.Println("Recv'd connect response")

		case shared.ChannelDisconnect:
			f.bayeuxState = stateBayeuxDisconnected
			f.disconnectFromServer()

		case shared.ChannelSubscribe:
			f.updateSubscription(fm.Subscription, fm.Successful)

		case shared.ChannelUnsubscribe:
			if fm.Successful {
				f.removeSubscription(fm.Subscription)
			}
		default:
			if fm.Data != nil {
				if fm.ClientID == f.clientID {
					return nil
				}
				var data map[string]interface{}
				var ext map[string]interface{}

				if fm.Data != nil {
					data = fm.Data.(map[string]interface{})
				}

				if fm.Ext != nil {
					ext = fm.Ext.(map[string]interface{})
				}

				// tell the client we got a message on a channel
				go func(d, e map[string]interface{}) {
					select {
					case f.MessageChan <- Message{Channel: fm.Channel, Data: d, Ext: e}:
						return
					case <-time.After(100 * time.Millisecond):
						return
					}
				}(data, ext)
			}
		}
	}

	return nil
}

// Subscribe sends a subscribe message for the given channel
func (f *Client) Subscribe(channel string) error {
	if len(channel) == 0 {
		return errors.New("Channel must have a value.")
	}
	f.addSubscription(channel)
	return f.subscribe(channel)
}

// Unsubscribe sends an unsubscribe message for the given channel
func (f *Client) Unsubscribe(channel string) error {
	if len(channel) == 0 {
		return errors.New("Channel must have a value.")
	}
	return f.unsubscribe(channel)
}

// Publish publishes a message to the Bayeux server
func (f *Client) Publish(channel string, data map[string]interface{}) error {
	return f.publish(channel, data)
}

// Disconnect disconnects from the Bayeux server
func (f *Client) Disconnect() {
	f.disconnect()
}

/*
Bayeux protocol messages
*/

/*
type BayeuxResponse struct {
	Channel                  string            `json:"channel,omitempty"`
	Successful               bool              `json:"successful,omitempty"`
	Version                  string            `json:"version,omitempty"`
	SupportedConnectionTypes []string          `json:"supportedConnectionTypes,omitempty"`
	ClientID                 string            `json:"clientID,omitempty"`
	Advice                   map[string]string `json:"advice,omitempty"`
	Subscription             string            `json:"subscription,omitempty"`
	Error                    string            `json:"error,omitempty"`
	Id                       string            `json:"id,omitempty"`
	Data                     interface{}       `json:"data,omitempty"`
}
*/

// Bayeux message functions

/*
 */
func (f *Client) handshake() {
	message := server.BayeuxResponse{Channel: shared.ChannelHandshake, Version: "1.0", SupportedConnectionTypes: []string{"websocket"}}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating handshake message")
	}
}

// connect to Bayeux and send the connect message
func (f *Client) connect() {
	message := server.BayeuxResponse{Channel: shared.ChannelConnect, ClientID: f.clientID, ConnectionType: "websocket"}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating connect message")
	}
}

// disconnect sends the disconnect message to the server
func (f *Client) disconnect() {
	message := server.BayeuxResponse{Channel: shared.ChannelDisconnect, ClientID: f.clientID}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating connect message")
	}
}

// subscribe the client to a channel
func (f *Client) subscribe(channel string) error {
	message := server.BayeuxResponse{Channel: shared.ChannelSubscribe, ClientID: f.clientID, Subscription: channel}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating subscribe message")
		return err
	}
	return nil
}

// unsubscribe from a channel
func (f *Client) unsubscribe(channel string) error {
	message := server.BayeuxResponse{Channel: shared.ChannelUnsubscribe, ClientID: f.clientID, Subscription: channel}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating unsubscribe message")
		return err
	}
	return nil
}

// publish sends a message to a channel
func (f *Client) publish(channel string, data map[string]interface{}) error {
	message := server.BayeuxResponse{Channel: channel, ClientID: f.clientID, ID: f.core.NextMessageID(), Data: data}
	err := f.writeMessage(message)
	if err != nil {
		fmt.Println("Error generating unsubscribe message")
		return err
	}
	return nil
}

// writeMessage encodes the json and send the message over the wire.
func (f *Client) writeMessage(message server.BayeuxResponse) error {
	if !f.conn.Connected() {
		// reconnect
		fmt.Println("RECONNECT")
		cerr := f.connectToServer()
		if cerr != nil {
			return cerr
		}
		if !f.conn.Connected() {
			errors.New("Not Connected, Reconnect Failed.")
		}

	}

	json, err := json.Marshal(message)
	if err != nil {
		return err
	}
	f.Write(string(json))
	return nil
}
