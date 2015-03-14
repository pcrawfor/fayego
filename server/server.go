// Package server implements the faye specific message handling and server logic for a faye backend server
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/pcrawfor/fayego/shared"
)

// Server is the main faye server object which manages connections, subscriptions and clients
type Server struct {
	Connections   []Connection
	Subscriptions map[string][]Client
	SubMutex      sync.RWMutex
	Clients       map[string]Client
	ClientMutex   sync.RWMutex
	core          shared.Core
}

// NewServer returns and instance of Server
func NewServer() *Server {
	return &Server{Connections: []Connection{},
		Subscriptions: make(map[string][]Client),
		Clients:       make(map[string]Client),
		core:          shared.Core{}}
}

// publishToChannel publishes the message data string to the given channel
func (f *Server) publishToChannel(channel, data string) {
	subs, ok := f.Subscriptions[channel]
	fmt.Println("Subs: ", f.Subscriptions, "count: ", len(f.Subscriptions[channel]))
	if ok {
		f.multiplexWrite(subs, data)
	}
}

// multiplexWrite sends data to the subscriptions provided
func (f *Server) multiplexWrite(subs []Client, data string) {
	var group sync.WaitGroup
	for i := range subs {
		fmt.Println("subs[i]: ", subs[i])
		group.Add(1)
		go func(client chan<- []byte, data string) {
			if client != nil {
				fmt.Println("WRITE FOR CLIENT")
				client <- []byte(data)
			} else {
				fmt.Println("NO CHANNEL DON'T TRY TO WRITE")
			}
			group.Done()
		}(subs[i].WriteChannel, data)
	}
	group.Wait()
}

// findClientForChannel looks up the client associated with a given write channel
func (f *Server) findClientForChannel(c chan []byte) *Client {
	f.ClientMutex.Lock()
	defer f.ClientMutex.Unlock()

	for _, client := range f.Clients {
		if client.WriteChannel == c {
			fmt.Println("Matched Client: ", client.ClientID)
			return &client
		}
	}
	return nil
}

// DisconnectChannel disconnects from the given channel sending the faye disconnect message
func (f *Server) DisconnectChannel(c chan []byte) {
	client := f.findClientForChannel(c)
	if client != nil {
		fmt.Println("Disconnect Client: ", client.ClientID)
		f.removeClientFromServer(client.ClientID)
	}
}

// FayeMessage represents an incoming Faye message along with the associated fields
type FayeMessage struct {
	Channel                  string      `json:"channel"`
	ClientID                 string      `json:"clientId,omitempty"`
	Subscription             string      `json:"subscription,omitempty"`
	Data                     interface{} `json:"data,omitempty"`
	ID                       string      `json:"id,omitempty"`
	SupportedConnectionTypes []string    `json:"supportedConnectionTypes,omitempty"`
}

// HandleMessage is the core Faye message handler for the server, it interprets each faye message and triggers the appropriate response
func (f *Server) HandleMessage(message []byte, c chan []byte) ([]byte, error) {
	fmt.Println("Raw message:", string(message))

	// parse message JSON
	fm := FayeMessage{}
	err := json.Unmarshal(message, &fm)

	if err != nil {
		fmt.Println("Error parsing message json, try array parse:", err)

		ar := []FayeMessage{}
		jerr := json.Unmarshal(message, &ar)
		if jerr != nil {
			fmt.Println("Error parsing message json as array:", err)
		} else {
			fm = ar[0]
			fmt.Println("Parsed as: ", fm)
		}
	}

	switch fm.Channel {
	case shared.ChannelHandshake:
		fmt.Println("handshake")
		return f.handshake()
	case shared.ChannelConnect:
		fmt.Println("connect")
		return f.connect(fm.ClientID)
	case shared.ChannelDisconnect:
		fmt.Println("disconnect")
		return f.disconnect(fm.ClientID)
	case shared.ChannelSubscribe:
		fmt.Println("subscribe")
		return f.subscribe(fm.ClientID, fm.Subscription, c)
	case shared.ChannelUnsubscribe:
		fmt.Println("subscribe")
		return f.unsubscribe(fm.ClientID, fm.Subscription)
	default:
		fmt.Println("publish")
		fmt.Println("data is: ", fm.Data)
		return f.publish(fm.Channel, fm.ID, fm.Data)
	}
}

// FayeResponse represents a Faye response message and all associated fields
type FayeResponse struct {
	Channel                  string                 `json:"channel,omitempty"`
	Successful               bool                   `json:"successful,omitempty"`
	Version                  string                 `json:"version,omitempty"`
	SupportedConnectionTypes []string               `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string                 `json:"connectionType,omitempty"`
	ClientID                 string                 `json:"clientId,omitempty"`
	Advice                   map[string]interface{} `json:"advice,omitempty"`
	Subscription             string                 `json:"subscription,omitempty"`
	Error                    string                 `json:"error,omitempty"`
	ID                       string                 `json:"id,omitempty"`
	Data                     interface{}            `json:"data,omitempty"`
	Ext                      interface{}            `json:"ext,omitempty"`
}

/*

Handshake:

Example response:
{
    "channel": "/meta/handshake",
    "successful": true,
    "version": "1.0",
    "supportedConnectionTypes": [
        "long-polling",
        "cross-origin-long-polling",
        "callback-polling",
        "websocket",
        "eventsource",
        "in-process"
    ],
    "clientId": "1fg1b9s10zm29e0ahpk490mzkqk3",
    "advice": {
        "reconnect": "retry",
        "interval": 0,
        "timeout": 45000
    }
}

Bayeux Handshake response

*/

func (f *Server) handshake() ([]byte, error) {
	fmt.Println("handshake!")

	// build response
	resp := FayeResponse{
		ID:                       "1",
		Channel:                  "/meta/handshake",
		Successful:               true,
		Version:                  "1.0",
		SupportedConnectionTypes: []string{"websocket", "callback-polling", "long-polling", "cross-origin-long-polling", "eventsource", "in-process"},
		ClientID:                 f.core.GenerateClientID(),
		Advice:                   map[string]interface{}{"reconnect": "retry", "interval": 0, "timeout": 45000},
	}

	// wrap it in an array & convert to json
	return json.Marshal([]FayeResponse{resp})
}

/*

Connect:

Example response
[
  {
     "channel": "/meta/connect",
     "successful": true,
     "error": "",
     "clientId": "Un1q31d3nt1f13r",
     "timestamp": "12:00:00 1970",
     "advice": { "reconnect": "retry" }
   }
]
*/

func (f *Server) connect(clientID string) ([]byte, error) {
	// TODO: setup client connection state

	resp := FayeResponse{
		Channel:    "/meta/connect",
		Successful: true,
		Error:      "",
		ClientID:   clientID,
		Advice:     map[string]interface{}{"reconnect": "retry"},
	}

	// wrap it in an array & convert to json
	return json.Marshal([]FayeResponse{resp})
}

/*
Disconnect

Example response
[
  {
     "channel": "/meta/disconnect",
     "clientId": "Un1q31d3nt1f13r"
     "successful": true
  }
]
*/

func (f *Server) disconnect(clientID string) ([]byte, error) {
	// tear down client connection state
	f.removeClientFromServer(clientID)

	resp := FayeResponse{
		Channel:    "/meta/disconnect",
		Successful: true,
		ClientID:   clientID,
	}

	// wrap it in an array & convert to json
	return json.Marshal([]FayeResponse{resp})
}

/*
Subscribe

Example response
[
  {
     "channel": "/meta/subscribe",
     "clientId": "Un1q31d3nt1f13r",
     "subscription": "/foo/**",
     "successful": true,
     "error": ""
   }
]
*/

func (f *Server) subscribe(clientID, subscription string, c chan []byte) ([]byte, error) {

	// subscribe the client to the given channel
	if len(subscription) == 0 {
		return []byte{}, errors.New("Subscription channel not present")
	}

	f.addClientToSubscription(clientID, subscription, c)

	// if successful send success response
	resp := FayeResponse{
		Channel:      "/meta/subscribe",
		ClientID:     clientID,
		Subscription: subscription,
		Successful:   true,
		Error:        "",
	}

	// TODO: handle failure case

	// wrap it in an array and convert to json
	return json.Marshal([]FayeResponse{resp})
}

/*
Unsubscribe

Example response
[
  {
     "channel": "/meta/unsubscribe",
     "clientId": "Un1q31d3nt1f13r",
     "subscription": "/foo/**",
     "successful": true,
     "error": ""
   }
]
*/

func (f *Server) unsubscribe(clientID, subscription string) ([]byte, error) {
	// TODO: unsubscribe the client from the given channel
	if len(subscription) == 0 {
		return []byte{}, errors.New("Subscription channel not present")
	}

	// remove the client as a subscriber on the channel
	if f.removeClientFromSubscription(clientID, subscription) {
		fmt.Println("Successful unsubscribe")
	} else {
		fmt.Println("Failed to unsubscribe")
	}

	// if successful send success response
	resp := FayeResponse{
		Channel:      "/meta/unsubscribe",
		ClientID:     clientID,
		Subscription: subscription,
		Successful:   true,
		Error:        "",
	}

	// TODO: handle failure case

	// wrap it in an array and convert to json
	return json.Marshal([]FayeResponse{resp})
}

/*
Publish

Example response
[
  {
     "channel": "/some/channel",
     "successful": true,
     "id": "some unique message id"
  }
]

*/
func (f *Server) publish(channel, id string, data interface{}) ([]byte, error) {

	//convert data back to json string
	message := FayeResponse{
		Channel: channel,
		ID:      id,
		Data:    data,
	}

	dataStr, err := json.Marshal([]FayeResponse{message})
	if err != nil {
		fmt.Println("Error parsing message!")
		return []byte{}, errors.New("Invalid Message Data")
	}
	fmt.Println("publish to: ", channel)
	fmt.Println("data: ", string(dataStr))

	f.publishToChannel(channel, string(dataStr))
	f.publishToWildcard(channel, string(dataStr))

	resp := FayeResponse{
		Channel:    channel,
		Successful: true,
		ID:         id,
	}

	return json.Marshal([]FayeResponse{resp})
}

func (f *Server) publishToWildcard(channel, dataStr string) {
	parts := strings.Split(channel, "/")
	parts = parts[:len(parts)-1]
	parts = append(parts, "*")
	wildcardChannel := strings.Join(parts, "/")
	fmt.Println("WILDCARD: ", wildcardChannel)
	f.publishToChannel(wildcardChannel, dataStr)
}
