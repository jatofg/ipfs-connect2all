package input

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/common/log"
	"ipfs-connect2all/helpers"
	"os"
	"strconv"
)

type VisitedPeer struct {
	NodeID peer.ID
	MultiAddrs []multiaddr.Multiaddr
	Reachable bool
	AgentVersion string
}

type visitedPeers_json struct {
	start_timestamp string
	end_timestamp string
	Nodes []visitedPeer_json
}

type visitedPeer_json struct {
	NodeID string
	MultiAddrs []string
	reachable bool
	agent_version string
}

type ConnectedPeer struct {
	NodeID peer.ID
	Direction network.Direction
	SupportedProtocols []protocol.ID
}

// load visitedPeers*.json file from an ipfs-crawler run
func LoadVisitedPeers(visitedPeersFile string) (map[peer.ID]*VisitedPeer, error) {
	f, err := os.Open(visitedPeersFile)
	if err != nil {
		return nil, errors.New("Could not open visitedPeers file for reading: " + err.Error())
	}

	r := json.NewDecoder(f)
	var jsonVisitedPeers visitedPeers_json
	err = r.Decode(&jsonVisitedPeers)
	if err != nil {
		return nil, errors.New("JSON decode error: " + err.Error())
	}

	ret := make(map[peer.ID]*VisitedPeer)
	for _, jsonVisitedPeer := range jsonVisitedPeers.Nodes {
		id, err := peer.IDFromString(jsonVisitedPeer.NodeID)
		if err != nil {
			log.Warnln("Could not decode peer ID from visitedPeers file: " + err.Error())
		}
		multiaddrs := make([]multiaddr.Multiaddr, 0, len(jsonVisitedPeer.MultiAddrs))
		for _, ma := range(jsonVisitedPeer.MultiAddrs) {
			newMa, err := multiaddr.NewMultiaddr(ma)
			if err != nil {
				return nil, errors.New("Could not decode multiaddr from visitedPeers file: " + err.Error())
			}
			multiaddrs = append(multiaddrs, newMa)
		}
		ret[id] = &VisitedPeer{
			NodeID: id,
			MultiAddrs: multiaddrs,
			Reachable: jsonVisitedPeer.reachable,
			AgentVersion: jsonVisitedPeer.agent_version,
		}
	}
	return ret, nil
}

// load connected peers from snapshot (connected_*.csv file)
func LoadConnectedPeers(connectedPeersFile string) (map[peer.ID]*ConnectedPeer, error) {
	f, err := os.Open(connectedPeersFile)
	if err != nil {
		return nil, errors.New("Could not open connected peers file for reading: " + err.Error())
	}
	r := csv.NewReader(f)
	r.Comma = ';'
	row, err := r.Read()
	ret := make(map[peer.ID]*ConnectedPeer)
	for ; err == nil; row, err = r.Read() {
		if len(row) < 3 {
			return ret, errors.New("Invalid CSV row length in connected peers file (should be at least 3)")
		}

		id, err := peer.IDFromString(row[0])
		if err != nil {
			return nil, errors.New("Could not decode peer ID from connected peers file: " + err.Error())
		}

		direction, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, errors.New("Could not read direction from connected peers file: " + err.Error())
		}

		ret[id] = &ConnectedPeer{
			NodeID: id,
			Direction: network.Direction(direction),
			SupportedProtocols: helpers.SupportedProtocolsFromString(row[2]),
		}
	}
	return ret, nil
}

// load peer list from other snapshot files
func LoadPeerList(peerListFile string) (map[peer.ID]peer.ID, error) {
	f, err := os.Open(peerListFile)
	if err != nil {
		return nil, errors.New("Could not open peer list file for reading: " + err.Error())
	}

	scn := bufio.NewScanner(f)
	ret := make(map[peer.ID]peer.ID)
	for scn.Scan() {
		st := scn.Text()
		if len(st) < 1 {
			continue
		}
 		id, err := peer.IDFromString(st)
		if err != nil {
			return nil, errors.New("Could not convert peer ID from string while reading peer list file: " + err.Error())
		}
		ret[id] = id
	}
	return ret, nil
}

func VisitedPeersToAddrInfoMap(visitedPeers map[peer.ID]*VisitedPeer) map[peer.ID]*peer.AddrInfo {
	ret := make(map[peer.ID]*peer.AddrInfo)
	for peerID, visitedPeer := range visitedPeers {
		ret[peerID] = &peer.AddrInfo{
			ID: visitedPeer.NodeID,
			Addrs: visitedPeer.MultiAddrs,
		}
	}
	return ret
}
