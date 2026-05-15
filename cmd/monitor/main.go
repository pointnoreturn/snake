package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

var (
	stream     *meshtastic.ProtoStream
	dispatch   *meshtastic.Dispatch
	myNodeInfo *pb.MyNodeInfo
	nodeInfo   *pb.NodeInfo
	db         DB
	reporter   Reporter
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	db.Init(ctx)
	reporter.Init(ctx)

	handlers := meshtastic.ChainPacketHandlers(
		printPacket,
		db.HandlePacket,
		reporter.HandlePacket,
	)

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	var err error

	stream, myNodeInfo, nodeInfo, err = meshtastic.FindAndConnect(ctx, targetNode, time.Second*10, meshtastic.ConfigId_ConfigOnly, db.HandlePacket)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("Cannot find target node to connect: " + targetNode)
		}
		panic(err)
	}
	defer stream.Close()

	label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)
	fmt.Printf("Connected node: %s (!%x), pio %s\n", label, myNodeInfo.MyNodeNum, myNodeInfo.PioEnv)

	dispatch = meshtastic.NewDispatch(stream, 100, handlers)

	go db.Run(ctx)
	go reporter.Run(ctx)

	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("Critical error in Dispatch.Run()")
			panic(err)
		}
	}
}
