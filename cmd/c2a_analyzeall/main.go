package main

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"ipfs-connect2all/analysis"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/stats"
	"os"
	"sort"
	"time"
)

func calculateComparisons(crawlAndSnapshotFiles analysis.CrawlOrSnapshotFiles, dateFormat string, outDir string) {
	timestamps := crawlAndSnapshotFiles.GetTimestamps("visitedPeers_", dateFormat)

	addWholeResult := func(sf *stats.StatsFile, comparisonResult analysis.ComparisonResult) {
		sf.AddInts(comparisonResult.DhtPeers, comparisonResult.ReachableDhtPeers, comparisonResult.KnownPeers,
			comparisonResult.ConnectedPeers, comparisonResult.SuccessfulConnections, comparisonResult.FailedConnections,
			comparisonResult.DhtButNotKnown, comparisonResult.DhtButNotConnected,
			comparisonResult.DhtButNotSuccessful, comparisonResult.DhtButFailed, comparisonResult.KnownButNotDht,
			comparisonResult.ConnectedButNotDht, comparisonResult.ConnectedButNotDhtReachable,
			comparisonResult.SuccessfulButNotDht, comparisonResult.SuccessfulButNotDhtReachable)
	}

	// at the start of crawl
	sf0, err := stats.NewFile(outDir + "/comparison_0m.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf0.FlushAndClose()
	// 10 min after start of crawl
	sf10, err := stats.NewFile(outDir + "/comparison_10m.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf10.FlushAndClose()
	// 20 min after start of crawl
	sf20, err := stats.NewFile(outDir + "/comparison_20m.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf20.FlushAndClose()
	// 30 mion after start of crawl
	sf30, err := stats.NewFile(outDir + "/comparison_23m.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf30.FlushAndClose()

	for _, ts := range timestamps {
		{
			filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts, dateFormat)
			if err != nil {
				continue
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf0, comparisonResult)
		}

		{
			filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*10),
				dateFormat)
			if err != nil {
				continue
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf10, comparisonResult)
		}

		{
			filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*20),
				dateFormat)
			if err != nil {
				continue
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf20, comparisonResult)
		}

		{
			filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*30),
				dateFormat)
			if err != nil {
				continue
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf30, comparisonResult)
		}
	}
}

func calculateChurn(crawlAndSnapshotFiles analysis.CrawlOrSnapshotFiles, dateFormat string, outDir string) {
	// calculate:
	// - duration of a connection (mean, median)
	// - how many new peers are known, connected in each snapshot
	// - how many connections to peers are lost (previously connected, now not anymore)
	startOfConnection := make(map[peer.ID]time.Time)
	alreadyKnownPeers := make(map[peer.ID]peer.ID)
	// TODO directly write to stats file?
	newPeersKnown := make(map[time.Time]int)
	newPeersConnected := make(map[time.Time]int)
	connectionsLost := make(map[time.Time]int)
	connectionDurations := make(map[peer.ID]time.Duration)
	timestamps := crawlAndSnapshotFiles.GetTimestamps("known_", dateFormat)

	// write stats files for points in time: newly known, newly connected, connection lost
	sf, err := stats.NewFile(outDir + "/churn.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf.FlushAndClose()

	// analysis does not make sense with less than 2 timestamps
	if len(timestamps) < 2 {
		panic("Cannot calculate churn with less than 2 snapshots")
	}

	for _, ts := range timestamps {
		filesForAnalysis, err := analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, time.Time{}, ts, dateFormat)
		if err != nil {
			continue
		}
		mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
		if err != nil {
			panic(err.Error())
		}

		newPeersKnown[ts] = 0
		newPeersConnected[ts] = 0
		connectionsLost[ts] = 0

		// newly known peers
		for knownPeer := range mapsForAnalysis.KnownPeers {
			if _, alreadyKnown := alreadyKnownPeers[knownPeer]; !alreadyKnown {
				newPeersKnown[ts]++
			}
		}
		alreadyKnownPeers = mapsForAnalysis.KnownPeers

		// peers not connected anymore
		toDelete := make([]peer.ID, 0)
		for alreadyConnectedPeer, connStarted := range startOfConnection {
			if _, stillConnected := mapsForAnalysis.ConnectedPeers[alreadyConnectedPeer]; !stillConnected {
				toDelete = append(toDelete, alreadyConnectedPeer)
				connectionDurations[alreadyConnectedPeer] = ts.Sub(connStarted)
				connectionsLost[ts]++
			}
		}
		for _, td := range toDelete {
			delete(alreadyKnownPeers, td)
		}

		// newly connected peers
		for connectedPeer := range mapsForAnalysis.ConnectedPeers {
			if _, alreadyConnected := startOfConnection[connectedPeer]; !alreadyConnected {
				newPeersConnected[ts]++
				startOfConnection[connectedPeer] = ts
			}
		}

		sf.AddInts(newPeersKnown[ts], newPeersConnected[ts], connectionsLost[ts])
	}
	// peers still connected at the end
	endTs := timestamps[len(timestamps)-1]
	for stillConnectedPeer, connStarted := range startOfConnection {
		connectionDurations[stillConnectedPeer] = endTs.Sub(connStarted)
	}

	// mean and median connection duration
	connDurationSum := 0.0
	connDurSlice := make([]float64, 0, len(connectionDurations))
	for _, dur := range connectionDurations {
		durMinutes := dur.Minutes()
		connDurationSum += durMinutes
		connDurSlice = append(connDurSlice, durMinutes)
	}
	sort.Float64s(connDurSlice)
	fmt.Println("Please note: Precision is limited by the snapshot interval")
	fmt.Printf("Mean connection duration in minutes: %f\n", connDurationSum/float64(len(connectionDurations)))
	fmt.Printf("Median connection duration in minutes: %f\n", connDurSlice[len(connDurSlice)/2])
	fmt.Printf("Shortest connection duration in minutes: %f\n", connDurSlice[0])
	fmt.Printf("Longest connection duration in minutes: %f\n", connDurSlice[len(connDurSlice)-1])

}

func main() {

	// default config values
	var configValues = make(map[string]string)
	configValues["DateFormat"] = "06-01-02--15:04:05"
	configValues["DHTCrawlDir"] = "crawls"
	configValues["SnapshotDir"] = "snapshots"
	configValues["OutputDir"] = "analysis_result"
	configValues["SkipComparisons"] = ""
	configValues["SkipChurn"] = ""

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) {
		fmt.Println("Usage: c2a_analysis [options]\n\n" +

			"Options:\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +
			"DHTCrawlDir=<dir>         Directory in which the snapshots are located\n" +
			"                          (default: snapshots)\n" +
			"SnapshotDir=<dir>         Directory in which the crawl output files are located\n" +
			"                          (default: crawls)\n" +
			"OutputDir=<dir>           Directory to which the output files will be saved\n" +
			"                          (default: analysis_result)\n" +
			"SkipComparisons           Do not calculate comparisons\n" +
			"SkipChurn                 Do not calculate churn")
		return
	}

	// create output dir if it does not exist yet
	err := helpers.CheckOrCreateDir(configValues["OutputDir"])
	if err != nil {
		panic("Could not create output dir: " + err.Error())
	}

	// the numbers from c2a_analysis, but for all crawls
	// load list of candidate files
	crawlAndSnapshotFiles, err := analysis.GetCrawlAndSnapshotFiles(configValues["DHTCrawlDir"], configValues["SnapshotDir"])
	if err != nil {
		panic(err.Error())
	}
	dateFormat := configValues["DateFormat"]
	outDir := configValues["OutputDir"]

	if configValues["SkipComparisons"] == "" {
		calculateComparisons(crawlAndSnapshotFiles, dateFormat, outDir)
	}

	if configValues["SkipChurn"] == "" {
		calculateChurn(crawlAndSnapshotFiles, dateFormat, outDir)
	}

}
