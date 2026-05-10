package libsnake

import "github.com/pointnoreturn/snake/libradio"

type Connection struct {
	r        libradio.Radio
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
