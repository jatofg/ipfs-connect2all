package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/input"
	"ipfs-connect2all/stats"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {

	// default config values
	var configValues = make(map[string]string)
	configValues["ConnMgrType"] = "none"
	configValues["ConnMgrHighWater"] = "0"
	configValues["LogToStdout"] = ""
	configValues["StatsFile"] = "peersStat.dat"
	configValues["MeasureConnections"] = ""
	configValues["DHTPeers"] = ""
	configValues["DHTConnsPerSec"] = "5"
	configValues["Snapshots"] = ""
	configValues["DateFormat"] = "06-01-02--15:04:05"
	configValues["StatsInterval"] = "5s"
	configValues["SnapshotInterval"] = "10m"
	configValues["DHTCrawlInterval"] = ""
	configValues["DHTCrawlOut"] = "crawls"
	configValues["DHTPreImages"] = "precomputed_hashes/preimages.csv"
	configValues["DHTQueueSize"] = "64384"
	configValues["DHTCacheFile"] = "crawls/nodes.cache"

	// load config from command line and display help upon encountering bad options (including -h/--help/Help/help/...)
	if !helpers.LoadConfig(&configValues, os.Args[1:]) {
		fmt.Println("Usage: ipfs-connect2all [options]\n\n" +

			"General options:\n" +
			"Help                      Show this help message and quit\n" +
			"LogToStdout               Write stats to stdout\n" +
			"StatsFile=<file>          Write to stats file <file> (default: peersStat.dat)\n" +
			"StatsInterval=<dur>       Stats collecting interval (default: 5s) (units available: ms, s, m, h)\n" +
			"ConnMgrType=basic         Use basic IPFS connection manager (instead of none)\n" +
			"ConnMgrHighWater=<value>  Max. number of peers in IPFS conn. manager (default: 0)\n" +
			"DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)\n" +
			"MeasureConnections=<file> Track average connection time and write to <file> \n" +
			"                          (default: no tracking, reduces concurrency)\n\n" +

			"Snapshot options:\n" +
			"Snapshots=<dir>           Write snapshots of currently known/... peers to files in <dir> (no trailing /)\n" +
			"SnapshotInterval=<dur>    Snapshot interval (default: 10m)\n\n" +

			"DHT scan options:\n" +
			"DHTPeers=<file>           Load visited peers from DHT crawl from visitedPeers*.csv file <file>\n" +
			"DHTConnsPerSec=<value>    Initiate <value> connections to peers from DHT crawl per second (default: 5)\n\n" +

			"ipfs-crawler integration options:\n" +
			"DHTCrawlInterval=<dur>    Crawl the DHT automatically in intervals of <dur> (default: off)\n" +
			"DHTCrawlOut=<dir>         Directory for saving the output files of DHT crawls (default: crawl)\n" +
			"DHTPreImages=<file>       File with preimages for DHT crawls (default: precomputed_hashes/preimages.csv)\n" +
			"DHTQueueSize=<size>       Queue size for DHT crawls (default: 64384)\n" +
			"DHTCacheFile=<file>       Cache file for DHT crawls (default: crawls/nodes.cache; empty to disable caching)")
		return
	}

	// constraints
	if configValues["ConnMgrType"] == "basic" {
		fmt.Println("Running with basic connection manager")
	} else {
		configValues["ConnMgrType"] = "none"
	}
	connMgrHighWater, err := strconv.Atoi(configValues["ConnMgrHighWater"])
	if err != nil {
		connMgrHighWater = 0
	}
	dhtConnsPerSec, err := strconv.Atoi(configValues["DHTConnsPerSec"])
	if err != nil {
		dhtConnsPerSec = 5
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ipfs := helpers.InitIpfs(ctx, configValues["ConnMgrType"], connMgrHighWater)

	// set bootstrap nodes
	bootstrapNodes := []string{
		// IPFS Bootstrapper nodes.
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",

		// IPFS Cluster Pinning nodes
		"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
		"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
		"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
		"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",

		// TODO add more nodes, e.g., from DHT scan?
	}

	// manage connections to track them
	connectionsMutex := &sync.Mutex{}
	connectionsInitiated := make(map[peer.ID]bool)
	connectionsFailed := make(map[peer.ID]bool)
	connectionsEstablished := make(map[peer.ID]bool)
	connectionsSuccessful := make(map[peer.ID]bool)

	checkConnectionAndSetInitiated := func(peerID peer.ID) bool {
		connectionsMutex.Lock()
		defer connectionsMutex.Unlock()
		if connectionsInitiated[peerID] || connectionsFailed[peerID] {
			return false
		}
		connectionsInitiated[peerID] = true
		return true
	}

	setConnectionInitiated := func(peerID peer.ID) {
		connectionsMutex.Lock()
		connectionsInitiated[peerID] = true
		connectionsMutex.Unlock()
	}

	setConnectionFailed := func(peerID peer.ID) {
		connectionsMutex.Lock()
		delete(connectionsInitiated, peerID)
		delete(connectionsEstablished, peerID)
		connectionsFailed[peerID] = true
		connectionsMutex.Unlock()
	}

	setConnectionEstablished := func(peerID peer.ID) {
		connectionsMutex.Lock()
		delete(connectionsInitiated, peerID)
		delete(connectionsFailed, peerID)
		connectionsEstablished[peerID] = true
		connectionsSuccessful[peerID] = true
		connectionsMutex.Unlock()
	}

	countConnections := func() (int, int, int, int) {
		connectionsMutex.Lock()
		defer connectionsMutex.Unlock()
		return len(connectionsEstablished), len(connectionsFailed), len(connectionsInitiated), len(connectionsSuccessful)
	}

	// connect to bootstrap peers
	bootstrapPeerInfos, err := helpers.MakePeerAddrInfoMap(bootstrapNodes)
	if err != nil {
		panic("Could not read list of bootstrap peers: " + err.Error())
		return
	}
	go func() {
		var wg sync.WaitGroup
		wg.Add(len(bootstrapPeerInfos))
		for _, peerInfo := range bootstrapPeerInfos {
			go func(peerInfo *peer.AddrInfo) {
				defer wg.Done()
				setConnectionInitiated(peerInfo.ID)
				err := ipfs.Swarm().Connect(ctx, *peerInfo)
				if err != nil {
					log.Printf("Could not connect to bootstrap peer %s: %s", peerInfo.ID, err)
					setConnectionFailed(peerInfo.ID)
				}
			}(peerInfo)
		}
		wg.Wait()
	}()

	// duration measurement
	connDurations := make([]time.Duration, 0, 10)
	connDurationsSuccess := make([]time.Duration, 0, 10)
	connDurationsFailure := make([]time.Duration, 0, 10)
	connDurationsMutex := &sync.Mutex{}
	measureConnections := configValues["MeasureConnections"] != ""

	// function to attempt to connect to a node and track progress
	tryToConnect := func(peerInfo peer.AddrInfo) {
		if !checkConnectionAndSetInitiated(peerInfo.ID) {
			return
		}

		var startTime time.Time
		var connDuration time.Duration
		if measureConnections {
			startTime = time.Now()
		}

		err := ipfs.Swarm().Connect(ctx, peerInfo)

		if measureConnections {
			connDuration = time.Now().Sub(startTime)
		}

		if err == nil {
			setConnectionEstablished(peerInfo.ID)

			if measureConnections {
				connDurationsMutex.Lock()
				connDurations = append(connDurations, connDuration)
				connDurationsSuccess = append(connDurationsSuccess, connDuration)
				connDurationsMutex.Unlock()
			}
		} else {
			setConnectionFailed(peerInfo.ID)

			if measureConnections {
				connDurationsMutex.Lock()
				connDurations = append(connDurations, connDuration)
				connDurationsFailure = append(connDurationsFailure, connDuration)
				connDurationsMutex.Unlock()
			}
		}
	}

	tryToConnectWithWg := func(wg *sync.WaitGroup, peerInfo peer.AddrInfo) {
		tryToConnect(peerInfo)
		wg.Done()
	}

	// slowly insert peers from DHT scan, if requested
	if configValues["DHTPeers"] != "" {
		go func() {
			var wg sync.WaitGroup
			dhtPeers, err := input.LoadVisitedPeers(configValues["DHTPeers"])
			if err != nil {
				log.Printf("Error loading peers from DHT scan: %s", err)
			}
			if dhtPeers != nil {
				wg.Add(len(dhtPeers))
				connsLeft := dhtConnsPerSec
				for _, peerAddr := range dhtPeers {
					if connsLeft < 1 {
						time.Sleep(time.Second)
						connsLeft = dhtConnsPerSec
					}
					go tryToConnectWithWg(&wg, *peerAddr)
					connsLeft--
				}
			}
			wg.Wait()
		}()
	}

	// use ipfs-crawler to run DHT crawls if requested
	if configValues["DHTCrawlInterval"] != "" {
		go func() {
			var crawlActive sync.WaitGroup
			for {
				crawlActive.Add(1)
				go func() {
					var wg sync.WaitGroup
					dhtPeers := input.CrawlDHT(configValues, helpers.PeerAddrInfoMapToSlice(bootstrapPeerInfos))
					if dhtPeers != nil {
						wg.Add(len(dhtPeers))
						connsLeft := dhtConnsPerSec
						for _, peerAddr := range dhtPeers {
							if connsLeft < 1 {
								time.Sleep(time.Second)
								connsLeft = dhtConnsPerSec
							}
							go tryToConnectWithWg(&wg, *peerAddr)
							connsLeft--
						}
					}
					wg.Wait()
					crawlActive.Done()
				}()

				interval, err := time.ParseDuration(configValues["DHTCrawlInterval"])
				if err != nil {
					interval = time.Hour*1
				}
				time.Sleep(interval)
				crawlActive.Wait()
			}
		}()
	}

	// collect number of connected and known peers and mean durations every 5s, try to connect to known peers
	// write stats to log files
	go func() {
		var currentStat *stats.StatsFile
		var err error
		if configValues["LogToStdout"] == "1" {
			currentStat, err = stats.NewFileWithCallback(configValues["StatsFile"], func(row []float64) {
				log.Printf("known=%d connected=%d established=%d failed=%d initiated=%d successful=%d",
					int(row[0]), int(row[1]), int(row[2]), int(row[3]), int(row[4]), int(row[5]))
			})
		} else {
			currentStat, err = stats.NewFile(configValues["StatsFile"])
		}
		if err != nil {
			panic("Error: Could not open stats file, connect2all will not work! Debug: " + err.Error())
			return
		}

		var durationStat *stats.StatsFile
		if measureConnections {
			durationStat, err = stats.NewFile(configValues["MeasureConnections"])
			if err != nil {
				log.Printf("Error: Could not open duration stats file, connect2all will not collect duration stats! Debug: %s", err.Error())
				measureConnections = false
			}
		}

		for {
			sleepDuration, err := time.ParseDuration(configValues["StatsInterval"])
			if err != nil {
				sleepDuration = time.Second*5
			}
			time.Sleep(sleepDuration)
			knownPeers, err := ipfs.Swarm().KnownAddrs(ctx)
			if err != nil {
				log.Printf("failed to get list of known peers: %s", err)
			}
			connectedPeers, err := ipfs.Swarm().Peers(ctx)
			if err != nil {
				log.Printf("failed to get list of connected peers: %s", err)
			}
			manEstablished, manFailed, manInitiated, manSuccessful := countConnections()
			currentStat.AddInts(len(knownPeers), len(connectedPeers),
				manEstablished, manFailed, manInitiated, manSuccessful)

			if measureConnections {
				connDurationsMutex.Lock()
				durationStat.AddFloats(helpers.DurationSliceMean(connDurations, time.Millisecond),
					helpers.DurationSliceMean(connDurationsSuccess, time.Millisecond),
					helpers.DurationSliceMean(connDurationsFailure, time.Millisecond))
				connDurationsMutex.Unlock()
			}

			for peerID, peerAddr := range knownPeers {
				// check if already connected
				// TODO make this more efficient
				alreadyConnected := false
				for _, connInfo := range connectedPeers {
					for _, peerAddrE := range peerAddr {
						if connInfo.Address() == peerAddrE {
							alreadyConnected = true
							break
						}
					}
					if alreadyConnected {
						break
					}
				}

				if !alreadyConnected {
					go tryToConnect(peer.AddrInfo{ID: peerID, Addrs: peerAddr})
				}
			}
		}
	}()

	// write snapshots once every 10 minutes as CSV
	snapshotDir := configValues["Snapshots"]
	if snapshotDir != "" {
		go func() {
			sdStat, err := os.Stat(snapshotDir)
			if err != nil {
				err2 := os.MkdirAll(snapshotDir, 0755)
				if err2 != nil {
					log.Printf("Directory %s could neither be accessed (error: %s) nor created (error: %s), " +
						"not writing snapshots.", snapshotDir, err.Error(), err2.Error())
					return
				}
			} else {
				if !sdStat.IsDir() {
					log.Printf("%s is not a directory, not writing snapshots.", snapshotDir)
					return
				}
			}

			dateFormat := configValues["DateFormat"]

			for {
				sleepDuration, err := time.ParseDuration(configValues["SnapshotInterval"])
				if err != nil {
					sleepDuration = time.Minute*10
				}
				time.Sleep(sleepDuration)

				knownPeers, err := ipfs.Swarm().KnownAddrs(ctx)
				if err != nil {
					log.Printf("failed to get list of known peers: %s", err)
					continue
				}
				err = helpers.WriteToCsv("known", snapshotDir, dateFormat,
					helpers.TransformMAMapForCsv(knownPeers))
				if err != nil {
					log.Printf("failed to write list of known peers to file: %s", err)
					continue
				}

				connPeers, err := ipfs.Swarm().Peers(ctx)
				if err != nil {
					log.Printf("failed to get list of connected peers: %s", err)
					continue
				}
				err = helpers.WriteToCsv("connected", snapshotDir, dateFormat,
					helpers.TransformConnInfoSliceForCsv(connPeers))
				if err != nil {
					log.Printf("failed to write list of connected peers to file: %s", err)
					continue
				}

				connectionsMutex.Lock()
				connEstablishedSlice := helpers.TransformBoolMapForCsv(connectionsEstablished)
				connSuccessfulSlice := helpers.TransformBoolMapForCsv(connectionsSuccessful)
				connFailedSlice := helpers.TransformBoolMapForCsv(connectionsFailed)
				connectionsMutex.Unlock()

				err = helpers.WriteToCsv("established", snapshotDir, dateFormat, connEstablishedSlice)
				if err != nil {
					log.Printf("failed to write list of established connections to file: %s", err)
					continue
				}

				err = helpers.WriteToCsv("successful", snapshotDir, dateFormat, connSuccessfulSlice)
				if err != nil {
					log.Printf("failed to write list of successful connections to file: %s", err)
					continue
				}

				err = helpers.WriteToCsv("failed", snapshotDir, dateFormat, connFailedSlice)
				if err != nil {
					log.Printf("failed to write list of failed connections to file: %s", err)
					continue
				}

			}
		}()
	}

	// flush data to files every 30s and at the end
	go func() {
		for {
			time.Sleep(time.Second*30)
			errs := stats.FlushAll()
			for _, err := range errs {
				log.Printf("Stats flush error: %s", err.Error())
			}
		}
	}()
	defer stats.FlushAndCloseAll()

	log.Print("Press enter to stop...\n\n")
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadLine()

}
