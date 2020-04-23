package input

import (
	"encoding/csv"
	"errors"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"io"
	"os"
	"strings"
)

// load visitedPeers*.csv file from an ipfs-crawler run
func LoadVisitedPeers(visitedPeersFile string) (map[peer.ID]*peer.AddrInfo, error) {
	f, err := os.Open(visitedPeersFile)
	if err != nil {
		return nil, errors.New("could not open visitedPeers file for reading: " + err.Error())
	}
	r := csv.NewReader(f)
	r.Comma = ';'
	row, err := r.Read()
	ret := make(map[peer.ID]*peer.AddrInfo)
	for ; err == nil; row, err = r.Read() {
		if len(row) < 3 {
			return ret, errors.New("invalid CSV row length in visitedPeers file (should be at least 3)")
		}

		// ignore non-reachable peers
		if row[2] != "true" {
			continue
		}

		// split second column (list of multiaddrs)
		if row[1][0] != '[' && len(row[1]) < 2 {
			return ret, errors.New("invalid multiaddr list in visited peers file")
		}
		maRem := row[1][1:]
		for {
			nextIndex := strings.IndexByte(maRem, ' ')
			if nextIndex < 1 {
				nextIndex = strings.IndexByte(maRem, ']')
			}
			if nextIndex < 1 {
				break
			}

			newMaStr := maRem[:nextIndex]
			newMa, err := multiaddr.NewMultiaddr(newMaStr)
			if err != nil {
				return ret, errors.New("error parsing a multiaddr from visitedPeers file: " + err.Error())
			}

			peerOfMa := peer.ID(row[0])
			if _, exists := ret[peerOfMa]; !exists {
				ret[peerOfMa] = &peer.AddrInfo{}
				ret[peerOfMa].ID = peerOfMa
			}
			ret[peerOfMa].Addrs = append(ret[peerOfMa].Addrs, newMa)

			if len(maRem) > nextIndex+1 {
				maRem = maRem[nextIndex+1:]
			} else {
				break
			}
		}
	}
	if err != io.EOF {
		return ret, errors.New("an error occurred while reading visitedPeers file: " + err.Error())
	}

	return ret, nil
}
