package main

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"ipfs-connect2all/analysis"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/stats"
	"os"
	"sort"
	"sync"
	"time"
)

func calculateComparisons(wg *sync.WaitGroup, files analysis.CrawlOrSnapshotFiles, dateFormat string, outDir string) {
	defer wg.Done()
	timestamps := files.GetTimestamps("visitedPeers_", dateFormat)

	// sort timestamps
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

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
	sf30, err := stats.NewFile(outDir + "/comparison_30m.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sf30.FlushAndClose()

	for _, ts := range timestamps {
		var wg sync.WaitGroup
		wg.Add(4)

		go func() {
			defer wg.Done()
			filesForAnalysis, err := analysis.GetFilesForAnalysis(files, ts, ts, dateFormat)
			if err != nil {
				return
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf0, comparisonResult)
		}()

		go func() {
			defer wg.Done()
			filesForAnalysis, err := analysis.GetFilesForAnalysis(files, ts, ts.Add(time.Minute*10),
				dateFormat)
			if err != nil {
				return
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf10, comparisonResult)
		}()

		go func() {
			defer wg.Done()
			filesForAnalysis, err := analysis.GetFilesForAnalysis(files, ts, ts.Add(time.Minute*20),
				dateFormat)
			if err != nil {
				return
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf20, comparisonResult)
		}()

		go func() {
			defer wg.Done()
			filesForAnalysis, err := analysis.GetFilesForAnalysis(files, ts, ts.Add(time.Minute*30),
				dateFormat)
			if err != nil {
				return
			}
			mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
			if err != nil {
				panic(err.Error())
			}
			comparisonResult := analysis.CalculateComparisonResult(*mapsForAnalysis)
			addWholeResult(sf30, comparisonResult)
		}()

		wg.Wait()
	}
}

func calculateChurn(wg *sync.WaitGroup, files analysis.CrawlOrSnapshotFiles, dateFormat string, outDir string,
					skipTotal bool) {
	defer wg.Done()

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
	timestamps := files.GetTimestamps("known_", dateFormat)

	// sort timestamps
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	// write stats files for points in time: newly known, newly connected, connection lost
	sfChurn, err := stats.NewFile(outDir + "/churn.dat")
	if err != nil {
		panic(err.Error())
	}
	defer sfChurn.FlushAndClose()

	// write stats files for total
	var sfTotal *stats.StatsFile
	if !skipTotal {
		sfTotal, err = stats.NewFile(outDir + "/total.dat")
		if err != nil {
			panic(err.Error())
		}
		defer sfTotal.FlushAndClose()
	}

	addWholeResult := func(sf *stats.StatsFile, maps *analysis.MapsForAnalysis) {
		sf.AddInts(len(maps.KnownPeers), len(maps.ConnectedPeers), len(maps.EstablishedConnections),
			len(maps.FailedConnections), len(maps.SuccessfulConnections))
	}

	// analysis does not make sense with less than 2 timestamps
	if len(timestamps) < 2 {
		panic("Cannot calculate churn with less than 2 snapshots")
	}

	lastTimestamp := timestamps[len(timestamps)-1]
	for _, ts := range timestamps {
		filesForAnalysis, err := analysis.GetFilesForAnalysis(files, time.Time{}, ts, dateFormat)
		if err != nil {
			continue
		}
		mapsForAnalysis, err := analysis.GetMapsForAnalysis(*filesForAnalysis)
		if err != nil {
			panic(err.Error())
		}

		if !skipTotal {
			addWholeResult(sfTotal, mapsForAnalysis)
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
			delete(startOfConnection, td)
		}

		// newly connected peers
		for connectedPeer := range mapsForAnalysis.ConnectedPeers {
			if _, alreadyConnected := startOfConnection[connectedPeer]; !alreadyConnected {
				newPeersConnected[ts]++
				// do not record start of connection if this is the last snapshot
				if ts != lastTimestamp {
					startOfConnection[connectedPeer] = ts
				}
			}
		}

		sfChurn.AddInts(newPeersKnown[ts], newPeersConnected[ts], connectionsLost[ts])
	}
	// peers still connected at the end
	for stillConnectedPeer, connStarted := range startOfConnection {
		connectionDurations[stillConnectedPeer] = lastTimestamp.Sub(connStarted)
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
	configValues["SkipTotal"] = ""

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) {
		fmt.Println("Usage: c2a_analyzeall [options]\n\n" +

			"Options:\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +
			"DHTCrawlDir=<dir>         Directory in which the snapshots are located\n" +
			"                          (default: snapshots)\n" +
			"SnapshotDir=<dir>         Directory in which the crawl output files are located\n" +
			"                          (default: crawls)\n" +
			"OutputDir=<dir>           Directory to which the output files will be saved\n" +
			"                          (default: analysis_result)\n" +
			"SkipComparisons           Do not calculate comparisons\n" +
			"SkipChurn                 Do not calculate churn\n" +
			"SkipTotal                 Do not record total numbers")
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

	var wg sync.WaitGroup

	if configValues["SkipComparisons"] == "" {
		wg.Add(1)
		go calculateComparisons(&wg, crawlAndSnapshotFiles, dateFormat, outDir)
	}

	if configValues["SkipChurn"] == "" {
		wg.Add(1)
		go calculateChurn(&wg, crawlAndSnapshotFiles, dateFormat, outDir, configValues["SkipTotal"] == "1")
	}

	wg.Wait()

}
