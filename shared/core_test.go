package shared

import "testing"

func setup() Core {
	return Core{}
}

func TestNextID(t *testing.T) {
	c := setup()
	id := c.NextMessageID()
	if id != "1" {
		t.Error("Expected message id to be 1 got:", id)
	}
}

func TestGenerateClientID(t *testing.T) {
	c := setup()
	id := c.GenerateClientID()
	if len(id) != 36 {
		t.Error("Expected UUID length 36 got:", len(id))
	}
}
