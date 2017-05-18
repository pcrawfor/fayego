package shared

import (
	"strconv"

	"code.google.com/p/go-uuid/uuid"
)

const ChannelHandshake = "/meta/handshake"
const ChannelConnect = "/meta/connect"
const ChannelDisconnect = "/meta/disconnect"
const ChannelSubscribe = "/meta/subscribe"
const ChannelUnsubscribe = "/meta/unsubscribe"

// Core shared functions
type Core struct {
	idCount int64
}

// NextMessageID returns the next message id incremented from the current running count on the instance of Core
func (c *Core) NextMessageID() string {
	c.idCount++
	return strconv.FormatInt(c.idCount, 10)
}

// GenerateClientID generates a new UUID string
func (c *Core) GenerateClientID() string {
	return uuid.New()
}
