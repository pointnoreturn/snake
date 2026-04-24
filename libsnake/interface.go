package libsnake

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/lmatte7/gomesh"
	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

func GetSerialNodes(devices ...string) [][]string {
	out := make([][]string, len(devices))
	for i, d := range devices {
		out[i] = []string{d, d}
	}
	return out
}

func DiscoverNodes(ctx context.Context, timeout time.Duration) [][]string {
	fmt.Println("Discover Meshtastic nodes...")

	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nodes := make(map[string][]string)

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

			ip := ""
			if len(e.AddrIPv4) > 0 {
				ip = e.AddrIPv4[0].String()
			}

			nodes[ip] = []string{
				strings.Trim(e.HostName, "."),
				ip,
			}

		case <-timer.C:
			out := make([][]string, 0, len(nodes))
			for _, v := range nodes {
				out = append(out, v)
			}
			return out
		}
	}
}

func Connect(target string) (*Connection, error) {

	r := gomesh.Radio{}
	err := r.Init(target)
	if err != nil {
		return nil, err
	}

	c := &Connection{r: r, Endpoint: target}

	info, err := c.GetSelfInfo()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Failed to GetSelfInfo for %s: %v", target, err)
	}

	c.Label = GetNodeLabel(info)
	c.NodeId = fmt.Sprintf("!%x", info.Num)
	return c, nil
}

func (c *Connection) GetSelfInfo() (*pb.NodeInfo, error) {

	responses, err := c.r.GetRadioInfo()
	if err != nil {
		return nil, fmt.Errorf("GetRadioInfo() failed: %w", err)
	}

	//fmt.Printf("Feteched %d responses from %s\n", len(responses), ip)

	for _, response := range responses {
		// Return FIRST node info assuming FIRST == SELF
		if info := response.GetNodeInfo(); info != nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("Zero node infos from Radio.GetRadioInfo")
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
			return Connect(n[1])
		}
	}

	for _, n := range nodes {
		c, err := Connect(n[1])
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
