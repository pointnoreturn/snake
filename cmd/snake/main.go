package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
)

func main() {
	fmt.Println("Scanning Meshtastic nodes...")

	nodes := libsnake.DiscoverNodes(3 * time.Second)

	for _, n := range nodes {

		label := "unknown"
		nodeID := "unknown"

		info, err := libsnake.GetSelfInfo(n[1])

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get node info for %s/%s: %v\n", n[0], n[1], err)
			continue
		}

		if info != nil {
			label = libsnake.GetNodeLabel(info)
			nodeID = fmt.Sprintf("!%x", info.Num)
		}

		fmt.Printf("Host: %-25s IP: %-15s Label: %s, ID: %s\n", n[0], n[1], label, nodeID)
	}
}
