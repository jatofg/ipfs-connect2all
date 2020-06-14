package main

import (
	"fmt"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/input"
	"log"
	"os"
	"time"
)

func main() {

	// default config values
	var configValues = make(map[string]string)
	configValues["DateFormat"] = "06-01-02--15:04:05"
	configValues["Timestamp"] = ""
	configValues["DHTCrawlDir"] = "crawls"
	configValues["SnapshotDir"] = "snapshots"

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) || configValues["Timestamp"] == "" {
		fmt.Println("Usage: c2a_analysis [arguments]\n\n" +

			"Required arguments:\n" +
			"Timestamp=<value>         Timestamp to use (see DateFormat, next crawl and \n" +
			"                          snapshots from this point will be used)\n\n" +

			"Optional arguments:\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +

			"DHTCrawlDir=<dir>         Directory in which the snapshots are located\n" +
			"SnapshotDir=<dir>         Directory in which the crawl output files are located")
		return
	}

	// load list of candidate files
	dhtCrawlDir, err := os.Open(configValues["DHTCrawlDir"])
	if err != nil {
		panic("DHT crawl dir could not be opened: " + err.Error())
	}
	snapshotDir, err := os.Open(configValues["SnapshotDir"])
	if err != nil {
		panic("Snapshot dir could not be opened: " + err.Error())
	}
	dhtCrawlCandidates, err := dhtCrawlDir.Readdirnames(-1)
	if err != nil {
		panic("Contents of DHT crawl dir could not be fetched: " + err.Error())
	}
	snapshotCandidates, err := snapshotDir.Readdirnames(-1)
	if err != nil {
		panic("Contents of snapshot dir could not be fetched: " + err.Error())
	}

	// choose the closest ones
	dateFormat := configValues["DateFormat"]
	inputTimestamp, err := time.Parse(dateFormat, configValues["Timestamp"])
	if err != nil {
		panic("Could not parse timestamp: " + err.Error())
	}

	getClosestFile := func(startsWith string, candidates []string) string {
		var currentDistance time.Duration = -1
		ret := ""
		for _, currentFile := range candidates {
			datePos := len(startsWith)
			if len(currentFile) <= datePos || currentFile[0:datePos] != startsWith {
				continue
			}
			currentTimestamp, err := time.Parse(dateFormat, currentFile[datePos:datePos+len(dateFormat)])
			if err != nil {
				fmt.Printf("Could not parse date of file %s, ignoring (error: %s).", currentFile, err.Error())
				continue
			}
			duration := currentTimestamp.Sub(inputTimestamp)
			if duration >= 0 && (currentDistance < 0 || duration < currentDistance) {
				ret = currentFile
				currentDistance = duration
			}
			if currentDistance == 0 {
				break
			}
		}
		return ret
	}

	visitedPeersFile := getClosestFile("visitedPeers_", dhtCrawlCandidates)
	if visitedPeersFile == "" {
		panic("Error: No matching DHT crawl file found.")
	} else {
		fmt.Printf("Using DHT crawl file: %s\n", visitedPeersFile)
	}

	knownFile := getClosestFile("known_", snapshotCandidates)
	if knownFile == "" {
		panic("Error: No matching known peers snapshot file found.")
	} else {
		fmt.Printf("Using known peers snapshot file: %s\n", knownFile)
	}

	connectedFile := getClosestFile("connected_", snapshotCandidates)
	if connectedFile == "" {
		panic("Error: No matching connected peers snapshot file found.")
	} else {
		fmt.Printf("Using connected peers snapshot file: %s\n", connectedFile)
	}

	//establishedFile := getClosestFile("established_", snapshotCandidates)
	//if establishedFile == "" {
	//	panic("Error: No matching established connections snapshot file found.")
	//} else {
	//	fmt.Printf("Using established connections snapshot file: %s\n", establishedFile)
	//}

	successfulFile := getClosestFile("successful_", snapshotCandidates)
	if successfulFile == "" {
		panic("Error: No matching successful connections snapshot file found.")
	} else {
		fmt.Printf("Using successful connections snapshot file: %s\n", successfulFile)
	}

	failedFile := getClosestFile("failed_", snapshotCandidates)
	if failedFile == "" {
		panic("Error: No matching failed connections snapshot file found.")
	} else {
		fmt.Printf("Using failed connections snapshot file: %s\n", failedFile)
	}

	fmt.Println()

	sWithDir := func(in string) string {
		return configValues["SnapshotDir"] + "/" + in
	}
	dhtWithDir := func(in string) string {
		return configValues["DHTCrawlDir"] + "/" + in
	}

	// Peers found by DHT scan, but not by connect2all:
	dhtPeers, err := input.LoadVisitedPeers(dhtWithDir(visitedPeersFile))
	if err != nil {
		log.Panicf("DHT peers could not be loaded: %s", err.Error())
	}
	knownPeers, err := input.LoadPeerList(sWithDir(knownFile))
	if err != nil {
		log.Panicf("Known peers could not be loaded: %s", err.Error())
	}
	connectedPeers, err := input.LoadConnectedPeers(sWithDir(connectedFile))
	if err != nil {
		log.Panicf("Connected peers could not be loaded: %s", err.Error())
	}
	successfulConnections, err := input.LoadPeerList(sWithDir(successfulFile))
	if err != nil {
		log.Panicf("Successful connections could not be loaded: %s", err.Error())
	}
	failedConnections, err := input.LoadPeerList(sWithDir(failedFile))
	if err != nil {
		log.Panicf("Failed connections could not be loaded: %s", err.Error())
	}

	reachableDhtPeers := len(dhtPeers)
	dhtButNotKnown := 0
	dhtButNotConnected := 0
	dhtButNotSuccessful := 0
	dhtButFailed := 0
	for dhtPeerID, dhtPeer := range dhtPeers {
		if !dhtPeer.Reachable {
			reachableDhtPeers--
			continue
		}
		if _, inKnown := knownPeers[dhtPeerID]; !inKnown {
			dhtButNotKnown++
		}
		if _, inConnected := connectedPeers[dhtPeerID]; !inConnected {
			dhtButNotConnected++
		}
		if _, inSuccessful := successfulConnections[dhtPeerID]; !inSuccessful {
			dhtButNotSuccessful++
		}
		if _, inFailed := failedConnections[dhtPeerID]; inFailed {
			dhtButFailed++
		}
	}

	knownButNotDht := 0
	for peerID := range knownPeers {
		if _, inDht := dhtPeers[peerID]; !inDht {
			knownButNotDht++
		}
	}

	connectedButNotDht := 0
	connectedButNotDhtReachable :=0
	for peerID := range connectedPeers {
		if _, inDht := dhtPeers[peerID]; !inDht {
			connectedButNotDht++
		} else if !dhtPeers[peerID].Reachable {
			connectedButNotDhtReachable++
		}
	}

	successfulButNotDht := 0
	successfulButNotDhtReachable :=0
	for peerID := range successfulConnections {
		if _, inDht := dhtPeers[peerID]; !inDht {
			successfulButNotDht++
		} else if !dhtPeers[peerID].Reachable {
			successfulButNotDhtReachable++
		}
	}

	failedButDhtReachable := 0
	for peerID := range failedConnections {
		if _, inDht := dhtPeers[peerID]; inDht {
			failedButDhtReachable++
		}
	}


	fmt.Printf("Reachable DHT peers: %d (total: %d)\n", reachableDhtPeers, len(dhtPeers))
	fmt.Printf("Peers known: %d; connected: %d\n", len(knownPeers), len(connectedPeers))
	fmt.Printf("Connections successful: %d; failed: %d\n\n", len(successfulConnections), len(failedConnections))

	fmt.Printf("DHT-reachable, but not c2a-known: %d\n", dhtButNotKnown)
	fmt.Printf("DHT-reachable, but not c2a-connected: %d\n", dhtButNotConnected)
	fmt.Printf("DHT-reachable, but not c2a-successful: %d\n", dhtButNotSuccessful)
	fmt.Printf("DHT-reachable, but c2a-failed: %d\n", dhtButFailed)
	fmt.Printf("c2a-known, but not in DHT crawl: %d\n", knownButNotDht)
	fmt.Printf("c2a-connected, but not in DHT crawl: %d\n", connectedButNotDht)
	fmt.Printf("c2a-connected, but not marked as reachable in DHT crawl: %d\n", connectedButNotDhtReachable)
	fmt.Printf("c2a-successful, but not in DHT crawl: %d\n", successfulButNotDht)
	fmt.Printf("c2a-successful, but not DHT-reachable: %d\n", successfulButNotDhtReachable)
	fmt.Printf("c2a-failed, but DHT-reachable: %d\n", failedButDhtReachable)
	// neither known nor failed?


	// TODO unique IDs in certain intervals, stability of connections?

}