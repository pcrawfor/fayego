package server

import "testing"

func TestClient(t *testing.T) {
	c := Client{"", nil, []string{"foo"}}
	if !c.isSubscribed("foo") {
		t.Error("Client should think it's subscribed to foo")
	}

	if c.isSubscribed("bar") {
		t.Error("Client should think it's subscribed to bar")
	}
}

func TestAddingRemovingClient(t *testing.T) {
	s := NewServer()
	clientID := "foo"
	subscription := "bar"
	c := s.addClientToServer(clientID, subscription, nil)
	if !c.isSubscribed(subscription) {
		t.Error("Should be subscribed to foo")
	}

	_, ok := s.Clients[clientID]
	if !ok {
		t.Error("Client not in client list should have been added")
	}

	e := s.removeClientFromServer(clientID)
	if e != nil {
		t.Error("Error removing client: ", e.Error())
	}

	_, ok = s.Clients[clientID]
	if ok {
		t.Error("Client still in client list should have been removed")
	}
}

func TestAddingRemovingSubscription(t *testing.T) {
	s := NewServer()
	clientID := "foo"
	subscription := "bar"
	ok := s.addClientToSubscription(clientID, subscription, nil)
	if !ok {
		t.Error("Failed to add client to subscription")
	}

	_, sok := s.Subscriptions[subscription]
	if !sok {
		t.Error("Subscription not present should have been added")
	}

	rok := s.removeClientFromSubscription(clientID, subscription)
	if !rok {
		t.Error("Failed to remove client from subscription")
	}

	c, cok := s.Clients[clientID]
	if !cok {
		t.Error("Client still in client list should have been removed")
	}

	if c.isSubscribed(subscription) {
		t.Error("Client is still subscribed but should not be")
	}
}

func TestUpdatingClientChannel(t *testing.T) {
	c := make(chan []byte)

	s := NewServer()
	clientID := "foo"
	subscription := "bar"
	ok := s.addClientToSubscription(clientID, subscription, nil)
	if !ok {
		t.Error("Failed to add client to subscription")
	}

	s.UpdateClientChannel(clientID, c)

	client, cok := s.Clients[clientID]
	if !cok {
		t.Error("Client not in client list should have been added")
	}

	if client.WriteChannel != c {
		t.Error("Expected client channel to be", c, "but it is", client.WriteChannel)
	}
}
