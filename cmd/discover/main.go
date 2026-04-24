package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
)

func main() {

	nodes := libsnake.DiscoverNodes(context.Background(), 5*time.Second)

	for _, n := range nodes {
		c, err := libsnake.Connect(n[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect %s/%s: %v\n", n[0], n[1], err)
			continue
		}

		fmt.Printf("Host: %-25s IP: %-15s Label: %s, ID: %s\n", n[0], n[1], c.Label, c.NodeId)

		c.Close()
	}
}
