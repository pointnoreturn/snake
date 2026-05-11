package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
)

func main() {
	fmt.Println("Discover network services.")
	services := libsnake.DiscoverServices(context.Background(), 5*time.Second)

	for _, svc := range services {
		fmt.Printf("SERVICE: [%s], Hostname: %s, Args: %+v, I: %s\n", svc.Endpoint, svc.Entry.HostName, svc.Args, svc.Entry.Instance)
	}

	fmt.Println("Get meshtastic nodes")
	for _, n := range libsnake.GetMeshtasticNodes(services) {

		fmt.Printf("NODE: [%s] id=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)

		fmt.Print("connect...")
		c, err := libsnake.ConnectMeshtastic(n.Service.Endpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect %s (%s): %v\n", n.Service.Endpoint, n.Label, err)
			continue
		}
		fmt.Println("DONE" + ": " + c.Label + ", " + c.NodeId)

		c.Close()
		fmt.Println("disconnected.")
	}
}
