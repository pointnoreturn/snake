package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/monitor/libradios"
	"github.com/pointnoreturn/monitor/meshtastic"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()
	fmt.Println("Discover advertised meshtastic nodes on the network")

	// browse timeout to wait for node announces
	timeoutContext, cancel := context.WithTimeout(ctx, time.Second*7)
	defer cancel()

	// Channels for browsing
	bs := make(chan *libradios.Broadcast)
	bn := make(chan *meshtastic.BroadcastNode)

	// Pull observed nodes on the network to list
	allNodes := []*meshtastic.BroadcastNode{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeoutContext.Done():
				return
			case n := <-bn:
				if n == nil {
					break
				}
				allNodes = append(allNodes, n)
				fmt.Printf("- Node: [%s]\tid=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
			}
		}
	}()

	go libradios.BrowseBroadcasts(timeoutContext, bs)
	meshtastic.BrowseNodes(timeoutContext, bs, bn)

	fmt.Printf("Total %d meshtastic nodes. Try connect-and-disconnect...\n", len(allNodes))
	if len(allNodes) == 0 {
		panic("No nodes found")
	}

	for _, n := range allNodes {

		fmt.Printf("test %s...\n", n.Service.Endpoint)
		stream, myNodeInfo, nodeInfo, err := meshtastic.ConnectTCP(ctx, n.Service.Endpoint, meshtastic.ConfigId_ConfigOnly, nil)
		if stream != nil {
			stream.Close()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed %s (%s): %v\n", n.Service.Endpoint, n.Label, err)
			continue
		}

		label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)

		fmt.Printf("test OK: %s, !%x\n", label, myNodeInfo.MyNodeNum)
	}
}
