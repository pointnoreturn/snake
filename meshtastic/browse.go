package meshtastic

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pointnoreturn/monitor/libradios"
)

func BrowseNodes(ctx context.Context, inBroadcasts chan *libradios.Broadcast, outNode chan *BroadcastNode) {
	defer close(outNode)

	for {
		select {
		case <-ctx.Done():
			return
		case bs := <-inBroadcasts:
			if bs == nil {
				break
			}

			fmt.Printf("Service: %+v\n", bs)

			if bs.Entry == nil {
				continue
			} else if bs.Entry.Service != "_meshtastic._tcp" {
				fmt.Printf("DEBUG: Unknown service '%s' at %s (%s), ignore\n", bs.Entry.Service, bs.Endpoint, bs.Entry.HostName)
				continue
			}

			if bs.Entry.Domain != "local." {
				fmt.Fprintf(os.Stderr, "INFO: Domaion is '%s', not local at %s (%s)\n", bs.Entry.Domain, bs.Endpoint, bs.Entry.HostName)
			}

			hexId, hasId := bs.Args["id"]
			shortName, hasShortName := bs.Args["shortname"]
			if !hasId || len(hexId) != 9 {
				fmt.Fprintf(os.Stderr, "ERR: Service has no 'id' key at %s (%s), drop\n", bs.Endpoint, bs.Entry.HostName)
				continue
			} else if !hasShortName {
				fmt.Fprintf(os.Stderr, "ERR: Service has no 'shortname' key at %s (%s), drop\n", bs.Endpoint, bs.Entry.HostName)
				continue
			}

			hexId = strings.TrimPrefix(hexId, "!")

			nodeNum, err := strconv.ParseUint(hexId, 16, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERR: Cannot parse 'id' key value '%s' as HEX int32 at %s (%s), drop\n", hexId, bs.Endpoint, bs.Entry.HostName)
				continue
			}

			// short name emoji fix
			if hasShortName {
				shortName = fixMeshtasticShortname(shortName)
			}

			label := shortName
			hexSuffix := hexId[len(hexId)-4:]
			if len(label) == 0 {
				label = hexSuffix + "_" + hexSuffix
			} else {
				label += "_" + hexSuffix
			}

			outNode <- &BroadcastNode{
				Service:   bs,
				ShortName: shortName,
				NodeNum:   uint32(nodeNum),
				Label:     label,
			}
		}
	}
}
