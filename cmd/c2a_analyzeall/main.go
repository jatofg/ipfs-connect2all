package main

import (
	"fmt"
	"ipfs-connect2all/analysis"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/stats"
	"os"
	"time"
)

func main() {

	// default config values
	var configValues = make(map[string]string)
	configValues["DateFormat"] = "06-01-02--15:04:05"
	configValues["DHTCrawlDir"] = "crawls"
	configValues["SnapshotDir"] = "snapshots"
	configValues["OutputDir"] = "analysis_result"

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) {
		fmt.Println("Usage: c2a_analysis [options]\n\n" +

			"Options:\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +

			"DHTCrawlDir=<dir>         Directory in which the snapshots are located\n" +
			"SnapshotDir=<dir>         Directory in which the crawl output files are located\n" +
			"OutputDir=<dir>           Directory to which the output files will be saved")
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
	timestamps := crawlAndSnapshotFiles.GetTimestamps("visitedPeers_", dateFormat)
	outDir := configValues["OutputDir"]

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

		filesForAnalysis, err = analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*10),
			dateFormat)
		if err != nil {
			continue
		}
		mapsForAnalysis, err = analysis.GetMapsForAnalysis(*filesForAnalysis)
		if err != nil {
			panic(err.Error())
		}
		comparisonResult = analysis.CalculateComparisonResult(*mapsForAnalysis)
		addWholeResult(sf10, comparisonResult)

		filesForAnalysis, err = analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*20),
			dateFormat)
		if err != nil {
			continue
		}
		mapsForAnalysis, err = analysis.GetMapsForAnalysis(*filesForAnalysis)
		if err != nil {
			panic(err.Error())
		}
		comparisonResult = analysis.CalculateComparisonResult(*mapsForAnalysis)
		addWholeResult(sf20, comparisonResult)

		filesForAnalysis, err = analysis.GetFilesForAnalysis(crawlAndSnapshotFiles, ts, ts.Add(time.Minute*30),
			dateFormat)
		if err != nil {
			continue
		}
		mapsForAnalysis, err = analysis.GetMapsForAnalysis(*filesForAnalysis)
		if err != nil {
			panic(err.Error())
		}
		comparisonResult = analysis.CalculateComparisonResult(*mapsForAnalysis)
		addWholeResult(sf30, comparisonResult)
	}

	// for how long are peers known, connected, ...
	// churn (peers seen for first time, last time, ...)

}
