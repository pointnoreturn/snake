package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	var c *meshtastic.Client = connect(ctx, targetNode)
	defer c.Close()
	fmt.Printf("Connected to: %s (!%x) at %s\n", c.Label, c.MyNode.MyNodeNum, c.Port)

	var t *meshtastic.Dispatch = meshtastic.NewDispatch(&c.ProtoStream, 10, []meshtastic.PacketF{
		func(p *pb.FromRadio) {
			logPacket(p, c.MyNode.MyNodeNum)
		},
	})

	err := t.Run(ctx)

	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("Non-critical error: " + err.Error())
			return
		}

		fmt.Println("Critical error in Dispatch.Run()")
		panic(err)
	}
}
