package libsnake

import "github.com/lmatte7/gomesh"

type Connection struct {
	r        gomesh.Radio
	Endpoint string
	NodeId   string
	Label    string
}

func (c *Connection) Close() {
	c.r.Close()
}

func (c *Connection) String() string {
	if len(c.Label) == 0 {
		return c.Endpoint
	}
	return c.Label
}
