package libsnake

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/lmatte7/gomesh"
	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

func DiscoverNodes(timeout time.Duration) [][]string {
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func GetSelfInfo(ip string) (*pb.NodeInfo, error) {

	r := gomesh.Radio{}
	err := r.Init(ip)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	responses, err := r.GetRadioInfo()
	if err != nil {
		return nil, fmt.Errorf("GetRadioInfo() failed for connected %s: %w", ip, err)
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
