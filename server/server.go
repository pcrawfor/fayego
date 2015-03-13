// package server implements the faye specific message handling and server logic for a faye backend server
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/pcrawfor/fayego/shared"
)

type Server struct {
	Connections   []Connection
	Subscriptions map[string][]Client
	SubMutex      sync.RWMutex
	Clients       map[string]Client
	ClientMutex   sync.RWMutex
	idCount       int
	core          shared.Core
}

/*
Instantiate a new faye server
*/
func NewServer() *Server {
	return &Server{Connections: []Connection{},
		Subscriptions: make(map[string][]Client),
		Clients:       make(map[string]Client),
		core:          shared.Core{}}
}

// general message handling
/*

*/
func (f *Server) publishToChannel(channel, data string) {
	subs, ok := f.Subscriptions[channel]
	fmt.Println("Subs: ", f.Subscriptions, "count: ", len(f.Subscriptions[channel]))
	if ok {
		f.multiplexWrite(subs, data)
	}
}

/*

*/
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

func (f *Server) findClientForChannel(c chan []byte) *Client {
	f.ClientMutex.Lock()
	defer f.ClientMutex.Unlock()

	for _, client := range f.Clients {
		if client.WriteChannel == c {
			fmt.Println("Matched Client: ", client.ClientId)
			return &client
		}
	}
	return nil
}

func (f *Server) DisconnectChannel(c chan []byte) {
	client := f.findClientForChannel(c)
	if client != nil {
		fmt.Println("Disconnect Client: ", client.ClientId)
		f.removeClientFromServer(client.ClientId)
	}
}

// ========

type FayeMessage struct {
	Channel                  string      `json:"channel"`
	ClientId                 string      `json:"clientId,omitempty"`
	Subscription             string      `json:"subscription,omitempty"`
	Data                     interface{} `json:"data,omitempty"`
	Id                       string      `json:"id,omitempty"`
	SupportedConnectionTypes []string    `json:"supportedConnectionTypes,omitempty"`
}

// Message handling

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
		return f.connect(fm.ClientId)
	case shared.ChannelDisconnect:
		fmt.Println("disconnect")
		return f.disconnect(fm.ClientId)
	case shared.ChannelSubscribe:
		fmt.Println("subscribe")
		return f.subscribe(fm.ClientId, fm.Subscription, c)
	case shared.ChannelUnsubscribe:
		fmt.Println("subscribe")
		return f.unsubscribe(fm.ClientId, fm.Subscription)
	default:
		fmt.Println("publish")
		fmt.Println("data is: ", fm.Data)
		return f.publish(fm.Channel, fm.Id, fm.Data)
	}
}

/*
FayeResponse
*/

type FayeResponse struct {
	Channel                  string                 `json:"channel,omitempty"`
	Successful               bool                   `json:"successful,omitempty"`
	Version                  string                 `json:"version,omitempty"`
	SupportedConnectionTypes []string               `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string                 `json:"connectionType,omitempty"`
	ClientId                 string                 `json:"clientId,omitempty"`
	Advice                   map[string]interface{} `json:"advice,omitempty"`
	Subscription             string                 `json:"subscription,omitempty"`
	Error                    string                 `json:"error,omitempty"`
	Id                       string                 `json:"id,omitempty"`
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
		Id:                       "1",
		Channel:                  "/meta/handshake",
		Successful:               true,
		Version:                  "1.0",
		SupportedConnectionTypes: []string{"websocket", "callback-polling", "long-polling", "cross-origin-long-polling", "eventsource", "in-process"},
		ClientId:                 f.core.GenerateClientID(),
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

func (f *Server) connect(clientId string) ([]byte, error) {
	// TODO: setup client connection state

	resp := FayeResponse{
		Channel:    "/meta/connect",
		Successful: true,
		Error:      "",
		ClientId:   clientId,
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

func (f *Server) disconnect(clientId string) ([]byte, error) {
	// tear down client connection state
	f.removeClientFromServer(clientId)

	resp := FayeResponse{
		Channel:    "/meta/disconnect",
		Successful: true,
		ClientId:   clientId,
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

func (f *Server) subscribe(clientId, subscription string, c chan []byte) ([]byte, error) {

	// subscribe the client to the given channel
	if len(subscription) == 0 {
		return []byte{}, errors.New("Subscription channel not present")
	}

	f.addClientToSubscription(clientId, subscription, c)

	// if successful send success response
	resp := FayeResponse{
		Channel:      "/meta/subscribe",
		ClientId:     clientId,
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

func (f *Server) unsubscribe(clientId, subscription string) ([]byte, error) {
	// TODO: unsubscribe the client from the given channel
	if len(subscription) == 0 {
		return []byte{}, errors.New("Subscription channel not present")
	}

	// remove the client as a subscriber on the channel
	if f.removeClientFromSubscription(clientId, subscription) {
		fmt.Println("Successful unsubscribe")
	} else {
		fmt.Println("Failed to unsubscribe")
	}

	// if successful send success response
	resp := FayeResponse{
		Channel:      "/meta/unsubscribe",
		ClientId:     clientId,
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
		Id:      id,
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
		Id:         id,
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
