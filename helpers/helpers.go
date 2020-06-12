package helpers

import (
	"encoding/csv"
	"errors"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"ipfs-connect2all/input"
	"os"
	"strconv"
	"strings"
	"time"
)

func MakePeerAddrInfoMap(nodesStr []string) (map[peer.ID]*peer.AddrInfo, error) {
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(nodesStr))
	for _, addrStr := range nodesStr {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			return nil, err
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, err
		}
		pi, ok := peerInfos[addrInfo.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: addrInfo.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, addrInfo.Addrs...)
	}
	return peerInfos, nil
}

func PeerAddrInfoMapToSlice(pmap map[peer.ID]*peer.AddrInfo) []*peer.AddrInfo {
	peerInfoSlice := make([]*peer.AddrInfo, len(pmap));
	i := 0
	for _, pi := range pmap {
		peerInfoSlice[i] = pi
		i++
	}
	return peerInfoSlice
}

func DurationSliceMean(inp []time.Duration, unit time.Duration) float64 {
	if len(inp) > 0 {
		var totalDuration time.Duration
		for _, cDuration := range inp {
			totalDuration += cDuration
		}
		return float64(totalDuration) / float64(len(inp)) / float64(unit)
	}
	return 0
}

// returns false if unknown option has been encountered
func LoadConfig(configMap *map[string]string, args []string) bool {
	for _, arg := range args {
		eqPos := strings.IndexByte(arg, '=')
		if eqPos < 0 {
			if _, optExists := (*configMap)[arg]; !optExists {
				return false
			}
			(*configMap)[arg] = "1"
		} else {
			if _, optExists := (*configMap)[arg[:eqPos]]; !optExists {
				return false
			}
			(*configMap)[arg[:eqPos]] = arg[eqPos+1:]
		}
	}
	return true
}

func CheckOrCreateDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err2 := os.MkdirAll(path, 0755)
			if err2 != nil {
				return err2
			}
			return nil
		} else {
			return err
		}
	}
	if !stat.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}

func WriteToCsv(prefix string, snapshotDir string, dateFormat string, elements [][]string) error {
	formattedDate := time.Now().Format(dateFormat)
	filename := snapshotDir + "/" + prefix + "_" + formattedDate + ".csv"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	csvWriter.Comma = ';'
	err = csvWriter.WriteAll(elements)
	if err != nil {
		return err
	}
	return nil
}

func TransformSliceForCsv(in []string) [][]string {
	out := make([][]string, len(in))
	for i, e := range in {
		out[i] = make([]string, 1)
		out[i][0] = e
	}
	return out
}

func TransformMAMapForCsv(in map[peer.ID][]multiaddr.Multiaddr) [][]string {
	out := make([][]string, len(in))
	i := 0
	for e, _ := range in {
		out[i] = make([]string, 1)
		out[i][0] = e.String()
		i++
	}
	return out
}

func TransformBoolMapForCsv(in map[peer.ID]bool) [][]string {
	out := make([][]string, len(in))
	i := 0
	for e, _ := range in {
		out[i] = make([]string, 1)
		out[i][0] = e.String()
		i++
	}
	return out
}

func SupportedProtocolsToString(in []protocol.ID) string {
	inStr := protocol.ConvertToStrings(in)
	return strings.Join(inStr, ",")
}

func SupportedProtocolsFromString(in string) []protocol.ID {
	retStr := strings.Split(in, ",")
	return protocol.ConvertFromStrings(retStr)
}

func TransformConnInfoSliceForCsv(in []iface.ConnectionInfo) [][]string {
	out := make([][]string, len(in))
	for i, e := range in {
		out[i] = make([]string, 3)
		out[i][0] = e.ID().String()
		out[i][1] = strconv.Itoa(int(e.Direction()))
		eStreams, err := e.Streams()
		if err == nil {
			out[i][2] = SupportedProtocolsToString(eStreams)
		}
	}
	return out
}

func VisitedPeersToAddrInfoMap(visitedPeers map[peer.ID]*input.VisitedPeer) map[peer.ID]*peer.AddrInfo {
	ret := make(map[peer.ID]*peer.AddrInfo)
	for peerID, visitedPeer := range visitedPeers {
		ret[peerID] = &peer.AddrInfo{
			ID: visitedPeer.NodeID,
			Addrs: visitedPeer.MultiAddrs,
		}
	}
	return ret
}
