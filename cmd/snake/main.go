package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
	"github.com/pointnoreturn/snake/libweather"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var w libweather.WeatherProvider = tryMakeWeatherProvider()

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	var conn *libsnake.Connection = mustConnect(targetNode)
	fmt.Println("Connected to: " + conn.String())

	var t *libsnake.Telemeter = libsnake.NewTelemeter(conn, w)
	t.RunLoop(context.TODO())
}

func mustConnect(targetNode string) *libsnake.Connection {
	ip := net.ParseIP(targetNode) // try parse as IP address

	if ip != nil { // connect by IP address
		if c, err := libsnake.Connect(ip.String()); err != nil {
			panic(fmt.Errorf("Failed to connect to TCP '%s': %w", targetNode, err))
		} else {
			return c
		}
	} else if strings.Index(targetNode, "/") == 0 { // serial device is a path
		if c, err := libsnake.Connect(targetNode); err != nil {
			panic(fmt.Errorf("Failed to connect to serial device '%s': %w", targetNode, err))
		} else {
			return c
		}
	} else { // discover on LAN, using mDNS scan + connect
		nodes := libsnake.DiscoverNodes(context.Background(), 5*time.Second)
		c, err := libsnake.FindAndConnect(targetNode, nodes)
		if err != nil {
			panic(fmt.Errorf("Failed to connect using discovery for '%s': %w", targetNode, err))
		} else {
			return c
		}
	}
}
