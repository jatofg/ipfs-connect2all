package analysis

import (
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"ipfs-connect2all/input"
	"os"
	"time"
)

type CrawlOrSnapshotFile struct {
	Filename string
	Directory string
}

type CrawlOrSnapshotFiles []CrawlOrSnapshotFile

func (f CrawlOrSnapshotFile) GetPath() string {
	return f.Directory + "/" + f.Filename
}

type FilesForAnalysis struct {
	VisitedPeersFile *CrawlOrSnapshotFile
	KnownPeersFile *CrawlOrSnapshotFile
	ConnectedPeersFile *CrawlOrSnapshotFile
	EstablishedConnectionsFile *CrawlOrSnapshotFile
	SuccessfulConnectionsFile *CrawlOrSnapshotFile
	FailedConnectionsFile *CrawlOrSnapshotFile
}

type MapsForAnalysis struct {
	VisitedPeers map[peer.ID]*input.VisitedPeer
	KnownPeers map[peer.ID]peer.ID
	ConnectedPeers map[peer.ID]*input.ConnectedPeer
	EstablishedConnections map[peer.ID]peer.ID
	SuccessfulConnections map[peer.ID]peer.ID
	FailedConnections map[peer.ID]peer.ID
}

type ComparisonResult struct {
	DhtPeers int
	ReachableDhtPeers int
	KnownPeers int
	ConnectedPeers int
	SuccessfulConnections int
	FailedConnections int
	DhtButNotKnown int
	DhtButNotConnected int
	DhtButNotSuccessful int
	DhtButFailed int
	KnownButNotDht int
	ConnectedButNotDht int
	ConnectedButNotDhtReachable int
	SuccessfulButNotDht int
	SuccessfulButNotDhtReachable int
}

func GetCrawlAndSnapshotFiles(crawlPath string, snapshotPath string) (CrawlOrSnapshotFiles, error) {
	dhtCrawlDir, err := os.Open(crawlPath)
	if err != nil {
		return nil, fmt.Errorf("DHT crawl dir could not be opened: %s", err.Error())
	}
	snapshotDir, err := os.Open(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("Snapshot dir could not be opened: %s", err.Error())
	}
	dhtCrawlCandidates, err := dhtCrawlDir.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("Contents of DHT crawl dir could not be fetched: %s", err.Error())
	}
	snapshotCandidates, err := snapshotDir.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("Contents of snapshot dir could not be fetched: %s", err.Error())
	}
	ret := make([]CrawlOrSnapshotFile, 0, len(dhtCrawlCandidates) + len(snapshotCandidates))
	for _, dcc := range dhtCrawlCandidates {
		ret = append(ret, CrawlOrSnapshotFile{Filename: dcc, Directory: crawlPath})
	}
	for _, sc := range snapshotCandidates {
		ret = append(ret, CrawlOrSnapshotFile{Filename: sc, Directory: snapshotPath})
	}
	return ret, nil
}

func (candidates CrawlOrSnapshotFiles) GetTimestamps(startsWith string, dateFormat string) []time.Time {
	ret := make([]time.Time, 0, len(candidates)/2)
	for _, currentFile := range candidates {
		datePos := len(startsWith)
		currentFileName := currentFile.Filename
		if len(currentFileName) <= datePos || currentFileName[0:datePos] != startsWith {
			continue
		}
		currentTimestamp, err := time.Parse(dateFormat, currentFileName[datePos:datePos+len(dateFormat)])
		if err != nil {
			continue
		}
		ret = append(ret, currentTimestamp)
	}
	return ret
}

func (candidates CrawlOrSnapshotFiles) GetClosest(startsWith string, timestamp time.Time, dateFormat string) *CrawlOrSnapshotFile {
	var currentDistance time.Duration = -1
	var ret *CrawlOrSnapshotFile = nil
	for _, currentFile := range candidates {
		datePos := len(startsWith)
		currentFileName := currentFile.Filename
		if len(currentFileName) <= datePos || currentFileName[0:datePos] != startsWith {
			continue
		}
		currentTimestamp, err := time.Parse(dateFormat, currentFileName[datePos:datePos+len(dateFormat)])
		if err != nil {
			//fmt.Printf("Could not parse date of file %s, ignoring (error: %s).", currentFileName, err.Error())
			continue
		}
		duration := currentTimestamp.Sub(timestamp)
		if duration >= 0 && (currentDistance < 0 || duration < currentDistance) {
			ret = &CrawlOrSnapshotFile{}
			*ret = currentFile
			currentDistance = duration
		}
		if currentDistance == 0 {
			break
		}
	}
	return ret
}

func GetFilesForAnalysis(inputFiles CrawlOrSnapshotFiles, crawlTimestamp time.Time, snapshotTimestamp time.Time,
							dateFormat string) (*FilesForAnalysis, error) {

	var visitedPeersFile *CrawlOrSnapshotFile = nil
	if !crawlTimestamp.IsZero() {
		visitedPeersFile = inputFiles.GetClosest("visitedPeers_", crawlTimestamp, dateFormat)
		if visitedPeersFile == nil {
			return nil, errors.New("Error: No matching DHT crawl file found.")
		}
	}

	knownFile := inputFiles.GetClosest("known_", snapshotTimestamp, dateFormat)
	if knownFile == nil {
		return nil, errors.New("Error: No matching known peers snapshot file found.")
	}

	connectedFile := inputFiles.GetClosest("connected_", snapshotTimestamp, dateFormat)
	if connectedFile == nil {
		return nil, errors.New("Error: No matching connected peers snapshot file found.")
	}

	establishedFile := inputFiles.GetClosest("established_", snapshotTimestamp, dateFormat)
	if establishedFile == nil {
		return nil, errors.New("Error: No matching established connections snapshot file found.")
	}

	successfulFile := inputFiles.GetClosest("successful_", snapshotTimestamp, dateFormat)
	if successfulFile == nil {
		return nil, errors.New("Error: No matching successful connections snapshot file found.")
	}

	failedFile := inputFiles.GetClosest("failed_", snapshotTimestamp, dateFormat)
	if failedFile == nil {
		return nil, errors.New("Error: No matching failed connections snapshot file found.")
	}

	return &FilesForAnalysis{
		VisitedPeersFile:           visitedPeersFile,
		KnownPeersFile:             knownFile,
		ConnectedPeersFile:         connectedFile,
		EstablishedConnectionsFile: establishedFile,
		SuccessfulConnectionsFile:  successfulFile,
		FailedConnectionsFile:      failedFile,
	}, nil
}

func GetMapsForAnalysis(filesForAnalysis FilesForAnalysis) (*MapsForAnalysis, error) {

	// Peers found by DHT scan, but not by connect2all:
	visitedPeers := make(map[peer.ID]*input.VisitedPeer)
	if filesForAnalysis.VisitedPeersFile != nil {
		var err error
		visitedPeers, err = input.LoadVisitedPeers(filesForAnalysis.VisitedPeersFile.GetPath())
		if err != nil {
			return nil, fmt.Errorf("DHT peers could not be loaded: %s", err.Error())
		}
	}
	knownPeers, err := input.LoadPeerList(filesForAnalysis.KnownPeersFile.GetPath())
	if err != nil {
		return nil, fmt.Errorf("Known peers could not be loaded: %s", err.Error())
	}
	connectedPeers, err := input.LoadConnectedPeers(filesForAnalysis.ConnectedPeersFile.GetPath())
	if err != nil {
		return nil, fmt.Errorf("Connected peers could not be loaded: %s", err.Error())
	}
	successfulConnections, err := input.LoadPeerList(filesForAnalysis.SuccessfulConnectionsFile.GetPath())
	if err != nil {
		return nil, fmt.Errorf("Successful connections could not be loaded: %s", err.Error())
	}
	failedConnections, err := input.LoadPeerList(filesForAnalysis.FailedConnectionsFile.GetPath())
	if err != nil {
		return nil, fmt.Errorf("Failed connections could not be loaded: %s", err.Error())
	}
	establishedConnections, err := input.LoadPeerList(filesForAnalysis.EstablishedConnectionsFile.GetPath())
	if err != nil {
		return nil, fmt.Errorf("Established connections could not be loaded: %s", err.Error())
	}

	return &MapsForAnalysis{
		VisitedPeers: visitedPeers,
		KnownPeers: knownPeers,
		ConnectedPeers: connectedPeers,
		EstablishedConnections: establishedConnections,
		SuccessfulConnections: successfulConnections,
		FailedConnections: failedConnections,
	}, nil

}

func CalculateComparisonResult(maps MapsForAnalysis) ComparisonResult {
	var result ComparisonResult
	result.DhtPeers = len(maps.VisitedPeers)
	result.ReachableDhtPeers = result.DhtPeers
	result.KnownPeers = len(maps.KnownPeers)
	result.ConnectedPeers = len(maps.ConnectedPeers)
	result.SuccessfulConnections = len(maps.SuccessfulConnections)
	result.FailedConnections = len(maps.FailedConnections)

	for dhtPeerID, dhtPeer := range maps.VisitedPeers {
		if !dhtPeer.Reachable {
			result.ReachableDhtPeers--
			continue
		}
		if _, inKnown := maps.KnownPeers[dhtPeerID]; !inKnown {
			result.DhtButNotKnown++
		}
		if _, inConnected := maps.ConnectedPeers[dhtPeerID]; !inConnected {
			result.DhtButNotConnected++
		}
		if _, inSuccessful := maps.SuccessfulConnections[dhtPeerID]; !inSuccessful {
			result.DhtButNotSuccessful++
		}
		if _, inFailed := maps.FailedConnections[dhtPeerID]; inFailed {
			result.DhtButFailed++
		}
	}

	for peerID := range maps.KnownPeers {
		if _, inDht := maps.VisitedPeers[peerID]; !inDht {
			result.KnownButNotDht++
		}
	}

	for peerID := range maps.ConnectedPeers {
		if _, inDht := maps.VisitedPeers[peerID]; !inDht {
			result.ConnectedButNotDht++
		} else if !maps.VisitedPeers[peerID].Reachable {
			result.ConnectedButNotDhtReachable++
		}
	}

	for peerID := range maps.SuccessfulConnections {
		if _, inDht := maps.VisitedPeers[peerID]; !inDht {
			result.SuccessfulButNotDht++
		} else if !maps.VisitedPeers[peerID].Reachable {
			result.SuccessfulButNotDhtReachable++
		}
	}

	return result
}

func CalculateDirections(connectedPeers map[peer.ID]*input.ConnectedPeer) (int, int) {
	inbound := 0
	outbound := 0
	for _, connectedPeer := range connectedPeers {
		switch connectedPeer.Direction {
		case network.DirInbound:
			inbound++
		case network.DirOutbound:
			outbound++
		}
	}
	return inbound, outbound
}
