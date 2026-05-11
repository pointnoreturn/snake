package libsnake

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/joho/godotenv"
	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libradio"
)

func GetSerialNodes(devices ...string) [][]string {
	out := make([][]string, len(devices))
	for i, d := range devices {
		out[i] = []string{d, d}
	}
	return out
}

func DiscoverServices(ctx context.Context, timeout time.Duration) []DiscoveredService {
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := []DiscoveredService{}

	go func() {
		_ = resolver.Browse(ctx, "_meshtastic._tcp", "local.", entries)
	}()

	timer := time.NewTimer(timeout)

	for {
		select {
		case e := <-entries:
			if e == nil {
				continue
			}

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

			services = append(services, DiscoveredService{
				Endpoint: endpoint,
				Entry:    e,
				Args:     args,
			})

		case <-timer.C:
			return services
		}
	}
}

func GetMeshtasticNodes(services []DiscoveredService) []MeshtasticNode {
	nodes := []MeshtasticNode{}
	for _, svc := range services {
		if svc.Entry == nil {
			continue
		} else if svc.Entry.Service != "_meshtastic._tcp" {
			fmt.Printf("DEBUG: Unknown service '%s' at %s (%s), ignore\n", svc.Entry.Service, svc.Endpoint, svc.Entry.HostName)
			continue
		}

		if svc.Entry.Domain != "local." {
			fmt.Fprintf(os.Stderr, "INFO: Domaion is '%s', not local at %s (%s)\n", svc.Entry.Domain, svc.Endpoint, svc.Entry.HostName)
		}

		if svc.Entry.Instance != "Meshtastic" {
			fmt.Fprintf(os.Stderr, "INFO: Service is '%s', not familiar Instance='%s'\n", svc.Endpoint, svc.Entry.Instance)
		}

		hexId, hasId := svc.Args["id"]
		shortName, hasShortName := svc.Args["shortname"]
		if !hasId || len(hexId) != 9 {
			fmt.Fprintf(os.Stderr, "ERR: Service has no 'id' key at %s (%s), drop\n", svc.Endpoint, svc.Entry.HostName)
			continue
		} else if !hasShortName {
			fmt.Fprintf(os.Stderr, "ERR: Service has no 'shortname' key at %s (%s), drop\n", svc.Endpoint, svc.Entry.HostName)
			continue
		}

		hexId = strings.TrimPrefix(hexId, "!")

		nodeNum, err := strconv.ParseUint(hexId, 16, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERR: Cannot parse 'id' key value '%s' as HEX int32 at %s (%s), drop\n", hexId, svc.Endpoint, svc.Entry.HostName)
			continue
		}

		label := shortName
		hexSuffix := hexId[len(hexId)-4:]
		if len(label) == 0 {
			label = hexSuffix + "_" + hexSuffix
		} else {
			label += "_" + hexSuffix
		}

		nodes = append(nodes, MeshtasticNode{
			Service:   svc,
			ShortName: shortName,
			NodeNum:   uint32(nodeNum),
			Label:     label,
		})
	}
	return nodes
}

func ConnectMeshtastic(target string) (*Connection, error) {

	r := libradio.Radio{}
	err := r.Init(target)
	if err != nil {
		return nil, err
	}

	c := &Connection{r: r, Endpoint: target}

	info, err := c.AdminGetSelfNode()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Failed to GetSelfInfo for %s: %v", target, err)
	}

	c.Label = GetNodeLabel(info)
	c.NodeId = fmt.Sprintf("!%x", info.Num)
	return c, nil
}

func (c *Connection) AdminGetSelfNode() (*pb.NodeInfo, error) {

	responses, err := c.r.GetRadioInfoBrief()
	if err != nil {
		return nil, fmt.Errorf("OwnerRequest() failed: %w", err)
	}

	fmt.Printf("Feteched %d responses from OwnerRequest() %s\n", len(responses), c.Endpoint)

	for _, response := range responses {
		// Return FIRST node info assuming FIRST == SELF
		if info := response.GetNodeInfo(); info != nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("Zero node infos from Radio.OwnerRequest")
}

func GetNodeLabel(info *pb.NodeInfo) string {

	short := info.User.ShortName
	nodeID := fmt.Sprintf("!%08x", info.Num)

	if len(nodeID) >= 6 && nodeID[0] == '!' {
		suffix := nodeID[len(nodeID)-4:]
		return fmt.Sprintf("%s_%s", strings.ToUpper(short), strings.ToUpper(suffix))
	} else if len(short) > 0 {
		return strings.ToUpper(short)
	}

	return fmt.Sprintf("!%x", info.Num)
}

func FindAndConnect(target string, nodes [][]string) (*Connection, error) {
	for _, n := range nodes {
		if strings.EqualFold(n[0], target) || n[1] == target { // match by host name or IP
			return ConnectMeshtastic(n[1])
		}
	}

	for _, n := range nodes {
		c, err := ConnectMeshtastic(n[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect %s/%s: %v\n", n[0], n[1], err)
			continue
		}

		if c.NodeId == target || c.Label == target { // match by label or node id
			return c, nil
		}

		c.Close()
	}

	return nil, fmt.Errorf("Failed to find node '%s' among %d nodes.", target, len(nodes))
}
