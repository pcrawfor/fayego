package server

import (
	"errors"
	"fmt"
)

// Client represents a connected Faye Client, each has an ID negotiated during handshake, a write channel tied to their network connection
// and a list of subscriptions(faye channels) that have been subscribed to by the client.
type Client struct {
	ClientID     string
	WriteChannel chan []byte
	ClientSubs   []string
}

// isSubscribed checks the subscription status for a given subscription name
func (c *Client) isSubscribed(sub string) bool {
	for _, clientSub := range c.ClientSubs {
		if clientSub == sub {
			return true
		}
	}
	return false
}

// subscriptionClientIndex finds the index for the given clientID in the subscriptions or returns -1 if not found
func (f *Server) subscriptionClientIndex(subscriptions []Client, clientID string) int {
	for i, c := range subscriptions {
		if c.ClientID == clientID {
			return i
		}
	}
	return -1
}

// removeSubFromClient removes a Subscription from a given client
func (f *Server) removeSubFromClient(client Client, sub string) Client {
	for i, clientSub := range client.ClientSubs {
		if clientSub == sub {
			client.ClientSubs = append(client.ClientSubs[:i], client.ClientSubs[i+1:]...)
			return client
		}
	}
	return client
}

// removeClientFromSubscription removes the given clientID from the subscription
func (f *Server) removeClientFromSubscription(clientID, subscription string) bool {
	fmt.Println("Remove Client From Subscription: ", subscription)

	// grab the client subscriptions array for the channel
	f.SubMutex.Lock()
	defer f.SubMutex.Unlock()

	subs, ok := f.Subscriptions[subscription]

	if !ok {
		return false
	}

	index := f.subscriptionClientIndex(subs, clientID)

	if index >= 0 {
		f.Subscriptions[subscription] = append(subs[:index], subs[index+1:]...)
	} else {
		return false
	}

	// remove sub from client subs list
	f.Clients[clientID] = f.removeSubFromClient(f.Clients[clientID], subscription)

	return true
}

// addClientFromSubscription adds the given clientID to the subscription
func (f *Server) addClientToSubscription(clientID, subscription string, c chan []byte) bool {
	fmt.Println("Add Client to Subscription: ", subscription)

	// Add client to server list if it is not present
	client := f.addClientToServer(clientID, subscription, c)

	// add the client as a subscriber to the channel if it is not already one
	f.SubMutex.Lock()
	defer f.SubMutex.Unlock()
	subs, cok := f.Subscriptions[subscription]
	if !cok {
		f.Subscriptions[subscription] = []Client{}
	}

	index := f.subscriptionClientIndex(subs, clientID)

	fmt.Println("Subs: ", f.Subscriptions, "count: ", len(f.Subscriptions[subscription]))

	if index < 0 {
		f.Subscriptions[subscription] = append(subs, *client)
		return true
	}

	return false
}

// client management

// UpdateClientChannel updates the write channel for the given clientID with the provided channel
func (f *Server) UpdateClientChannel(clientID string, c chan []byte) bool {
	fmt.Println("update client for channel: clientID: ", clientID)
	f.ClientMutex.Lock()
	defer f.ClientMutex.Unlock()
	client, ok := f.Clients[clientID]
	if !ok {
		client = Client{clientID, c, []string{}}
		f.Clients[clientID] = client
		return true
	}

	client.WriteChannel = c
	f.Clients[clientID] = client

	return true
}

// addClientToServer Add Client to server only if the client is not already present
func (f *Server) addClientToServer(clientID, subscription string, c chan []byte) *Client {
	fmt.Println("Add client: ", clientID)

	f.ClientMutex.Lock()
	defer f.ClientMutex.Unlock()
	client, ok := f.Clients[clientID]
	if !ok {
		client = Client{clientID, c, []string{}}
		f.Clients[clientID] = client
	}

	fmt.Println("Client subs: ", len(client.ClientSubs), " | ", client.ClientSubs)

	// add the subscription to the client subs list
	if !client.isSubscribed(subscription) {
		fmt.Println("Client not subscribed")
		client.ClientSubs = append(client.ClientSubs, subscription)
		f.Clients[clientID] = client
		fmt.Println("Client sub count: ", len(client.ClientSubs))
	} else {
		fmt.Println("Client already subscribed")
	}

	return &client
}

// removeClientFromServer temoves the Client from the server and unsubscribe from any subscriptions
func (f *Server) removeClientFromServer(clientID string) error {
	fmt.Println("Remove client: ", clientID)

	f.ClientMutex.Lock()
	defer f.ClientMutex.Unlock()

	client, ok := f.Clients[clientID]
	if !ok {
		return errors.New("Error removing client")
	}

	// clear any subscriptions
	for _, sub := range client.ClientSubs {
		fmt.Println("Remove sub: ", sub)
		if f.removeClientFromSubscription(client.ClientID, sub) {
			fmt.Println("Removed sub!")
		} else {
			fmt.Println("Failed to remove sub.")
		}
	}

	// remove the client from the server
	delete(f.Clients, clientID)

	return nil
}
