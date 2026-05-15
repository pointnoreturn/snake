package libradios

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/joho/godotenv"
)

func BrowseBroadcasts(ctx context.Context, outService chan *Broadcast) error {
	defer close(outService)

	entries := make(chan *zeroconf.ServiceEntry)

	resolver, _ := zeroconf.NewResolver(nil)
	go resolver.Browse(ctx, "_meshtastic._tcp", "local.", entries)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-entries:
			if e == nil {
				continue
			}

			fmt.Printf("Entry: %+v\n", e)

			endpoint := ""
			if len(e.AddrIPv4) > 0 {
				endpoint = fmt.Sprintf("%s:%d", e.AddrIPv4[0].String(), e.Port)
			} else if len(e.AddrIPv6) > 0 {
				endpoint = fmt.Sprintf("[%s]:%d", e.AddrIPv6[0].String(), e.Port)
			}

			// key=value pairs in Entry.Text
			args, err := godotenv.Unmarshal(strings.Join(e.Text, "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				args = make(map[string]string)
			}

			outService <- &Broadcast{
				Endpoint: endpoint,
				Entry:    e,
				Args:     args,
			}
		}
	}
}
