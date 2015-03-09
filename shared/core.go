package shared

import (
	"strconv"

	"code.google.com/p/go-uuid/uuid"
)

type Core struct {
	idCount int64
}

func (c *Core) NextMessageID() string {
	c.idCount++
	return strconv.FormatInt(c.idCount, 10)
}

func (c *Core) GenerateClientID() string {
	return uuid.New()
}
