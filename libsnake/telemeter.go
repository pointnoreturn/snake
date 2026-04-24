package libsnake

import (
	"context"
	"fmt"
	"time"
)

type Telemeter struct {
	conn *Connection
}

func NewTelemeter(conn *Connection) *Telemeter {
	return &Telemeter{conn: conn}
}

func (t *Telemeter) RunLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	fmt.Println("Telemeter loop is running")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.update()
		}
	}
}

func (t *Telemeter) update() {
	// TODO: generate telemetry
}
