package main

import (
	"fmt"
	"ipfs-connect2all/analysis"
	"ipfs-connect2all/helpers"
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
	configValues["SnapshotTS"] = ""

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) || configValues["Timestamp"] == "" {
		fmt.Println("Usage: c2a_analysis [arguments]\n\n" +

			"Required arguments:\n" +
			"Timestamp=<value>         Timestamp to use (see DateFormat, next crawl and\n" +
			"                          snapshots from this point will be used)\n\n" +

			"Optional arguments:\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +

			"DHTCrawlDir=<dir>         Directory in which the snapshots are located\n" +
			"SnapshotDir=<dir>         Directory in which the crawl output files are located\n" +
			"SnapshotTS=<value>        Timestamp to use for snapshots (if given, the one" +
			"                          from above will be used for the crawl only)")
		return
	}

	// load list of candidate files
	crawlAndSnapshotFiles, err := analysis.GetCrawlAndSnapshotFiles(configValues["DHTCrawlDir"], configValues["SnapshotDir"])
	if err != nil {
		panic(err.Error())
	}

	// choose the closest ones
	dateFormat := configValues["DateFormat"]
	inputTimestamp, err := time.Parse(dateFormat, configValues["Timestamp"])
	if err != nil {
		panic("Could not parse timestamp: " + err.Error())
	}
	var inputSnapshotTS time.Time
	if configValues["SnapshotTS"] != "" {
		inputSnapshotTS, err = time.Parse(dateFormat, configValues["SnapshotTS"])
		if err != nil {
			panic("Could not parse snapshot timestamp: " + err.Error())
		}
	} else {
		inputSnapshotTS = inputTimestamp
	}

	filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, inputTimestamp, inputSnapshotTS,
		dateFormat)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Using DHT crawl file: %s\n", filesForAnalysis.VisitedPeersFile)
	fmt.Printf("Using known peers snapshot file: %s\n", filesForAnalysis.KnownPeersFile)
	fmt.Printf("Using connected peers snapshot file: %s\n", filesForAnalysis.ConnectedPeersFile)
	fmt.Printf("Using established connections snapshot file: %s\n", filesForAnalysis.EstablishedConnectionsFile)
	fmt.Printf("Using successful connections snapshot file: %s\n", filesForAnalysis.SuccessfulConnectionsFile)
	fmt.Printf("Using failed connections snapshot file: %s\n", filesForAnalysis.FailedConnectionsFile)
	fmt.Println()

	mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
	if err != nil {
		panic(err.Error())
	}

	comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)

	fmt.Printf("Reachable DHT peers: %d (total: %d)\n", comparisonResult.ReachableDhtPeers,
		comparisonResult.DhtPeers)
	fmt.Printf("Peers known: %d; connected: %d\n", comparisonResult.KnownPeers, comparisonResult.ConnectedPeers)
	fmt.Printf("Connections successful: %d; failed: %d\n\n", comparisonResult.SuccessfulConnections,
		comparisonResult.FailedConnections)

	fmt.Printf("DHT-reachable, but not c2a-known: %d\n", comparisonResult.DhtButNotKnown)
	fmt.Printf("DHT-reachable, but not c2a-connected: %d\n", comparisonResult.DhtButNotConnected)
	fmt.Printf("DHT-reachable, but not c2a-successful: %d\n", comparisonResult.DhtButNotSuccessful)
	fmt.Printf("DHT-reachable, but c2a-failed: %d\n", comparisonResult.DhtButFailed)
	fmt.Printf("c2a-known, but not in DHT crawl: %d\n", comparisonResult.KnownButNotDht)
	fmt.Printf("c2a-connected, but not in DHT crawl: %d\n", comparisonResult.ConnectedButNotDht)
	fmt.Printf("c2a-connected, but not marked as reachable in DHT crawl: %d\n", comparisonResult.ConnectedButNotDhtReachable)
	fmt.Printf("c2a-successful, but not in DHT crawl: %d\n", comparisonResult.SuccessfulButNotDht)
	fmt.Printf("c2a-successful, but not DHT-reachable: %d\n", comparisonResult.SuccessfulButNotDhtReachable)
	// neither known nor failed?


	// TODO unique IDs in certain intervals, stability of connections?

}