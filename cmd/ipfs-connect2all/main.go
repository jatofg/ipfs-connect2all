package main

import (
	"bufio"
	"context"
	"fmt"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"ipfs-connect2all/helpers"
	"ipfs-connect2all/input"
	"ipfs-connect2all/stats"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var configValues = make(map[string]string)

// spawn node on temporary repository
func ipfsInit(ctx context.Context) iface.CoreAPI {

	// some of the initialization steps are taken from the example go-ipfs-as-a-library in the go-ipfs project

	// set up plugins
	plugins, err := loader.NewPluginLoader(filepath.Join("", "plugins"))
	if err != nil {
		panic(fmt.Errorf("error loading plugins: %s", err))
	}
	if err := plugins.Initialize(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}
	if err := plugins.Inject(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}

	// create temporary repo
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		panic(fmt.Errorf("failed to get temp dir: %s", err))
	}

	// Create config with a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		panic(err)
	}
	// custom config values
	cfg.Swarm.ConnMgr.Type = configValues["ConnMgrType"]
	cfg.Swarm.ConnMgr.HighWater, _ = strconv.Atoi(configValues["ConnMgrHighWater"])

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		panic(fmt.Errorf("failed to init ephemeral node: %s", err))
	}

	// Open repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		panic(err)
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}
	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		panic(err)
	}
	ipfs, err := coreapi.NewCoreAPI(node)
	if err != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
	}
	fmt.Println("IPFS node created successfully!")

	return ipfs
}

func main() {

	// set config from command line arguments
	configValues["ConnMgrType"] = "none"
	configValues["ConnMgrHighWater"] = "0"
	configValues["StatsFile"] = "peersStat.dat"
	configValues["DHTConnsPerSec"] = "5"
	for _, arg := range os.Args[1:] {
		if arg == "ConnMgrType=basic" {
			fmt.Println("Running with basic connection manager")
			configValues["ConnMgrType"] = "basic"
		} else if len(arg) > 17 && arg[:17] == "ConnMgrHighWater=" {
			_, err := strconv.Atoi(arg[17:])
			if err == nil {
				configValues["ConnMgrHighWater"] = arg[17:]
				fmt.Printf("Running with ConnMgrHighWater=%s\n", arg[17:])
			}
		} else if arg == "LogToStdout" {
			configValues["LogToStdout"] = "1"
		} else if len(arg) > 10 && arg[:10] == "StatsFile=" {
			configValues["StatsFile"] = arg[10:]
		} else if len(arg) > 19 && arg[:19] == "MeasureConnections=" {
			configValues["MeasureConnections"] = arg[19:]
		} else if len(arg) > 9 && arg[:9] == "DHTPeers=" {
			configValues["DHTPeers"] = arg[9:]
		} else if len(arg) > 15 && arg[:15] == "DHTConnsPerSec=" {
			_, err := strconv.Atoi(arg[15:])
			if err == nil {
				configValues["DHTConnsPerSec"] = arg[15:]
			}
		} else {
			if arg != "-h" && arg != "--help" && strings.ToLower(arg) != "help" {
				fmt.Printf("Unknown option %s, usage:\n", arg)
			} else {
				fmt.Println("Usage:")
			}
			fmt.Println("ipfs-connect2all [options]\n\n" +
				"Available options:\n" +
				"Help                      Show this help message and quit\n" +
				"LogToStdout               Write stats to stdout\n" +
				"StatsFile=<file>          Write to stats file <file> (default: peersStat.dat)\n" +
				"ConnMgrType=basic         Use basic IPFS connection manager (instead of none)\n" +
				"ConnMgrHighWater=<value>  Max. number of peers in IPFS conn. manager (default: 0)\n" +
				"MeasureConnections=<file> Track average connection time and write to <file> \n" +
				"                          (default: no tracking, reduces concurrency)\n" +
				"DHTPeers=<file>           Load visited peers from DHT scan from visitedPeers*.csv file <file>\n" +
				"DHTConnsPerSec=<value>    Initiate <value> connections to peers from DHT scan per second (default: 5)")
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ipfs := ipfsInit(ctx)

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
	go func() {
		var wg sync.WaitGroup
		peerInfos := make(map[peer.ID]*peer.AddrInfo, len(bootstrapNodes))
		for _, addrStr := range bootstrapNodes {
			addr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				return
			}
			addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				return
			}
			pi, ok := peerInfos[addrInfo.ID]
			if !ok {
				pi = &peer.AddrInfo{ID: addrInfo.ID}
				peerInfos[pi.ID] = pi
			}
			pi.Addrs = append(pi.Addrs, addrInfo.Addrs...)
		}

		wg.Add(len(peerInfos))
		for _, peerInfo := range peerInfos {
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
	_, measureConnections := configValues["MeasureConnections"]

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
	if _, useDht := configValues["DHTPeers"]; useDht {
		go func() {
			var wg sync.WaitGroup
			dhtPeers, err := input.LoadVisitedPeers(configValues["DHTPeers"])
			if err != nil {
				log.Printf("Error loading peers from DHT scan: %s", err)
			}
			if dhtPeers != nil {
				wg.Add(len(dhtPeers))
				connsPerSecond, _ := strconv.Atoi(configValues["DHTConnsPerSec"])
				connsLeft := connsPerSecond
				for _, peerAddr := range dhtPeers {
					if connsLeft < 1 {
						time.Sleep(time.Second)
						connsLeft = connsPerSecond
					}
					go tryToConnectWithWg(&wg, *peerAddr)
					connsLeft--
				}
			}
			wg.Wait()
		}()
	}

	// collect number of connected and known peers and mean durations every 5s, try to connect to known peers
	// write stats to log files
	go func() {
		var currentStat *stats.StatsFile
		if configValues["LogToStdout"] == "1" {
			currentStat = stats.NewFileWithCallback(configValues["StatsFile"], func(row []float64) {
				log.Printf("known=%d connected=%d established=%d failed=%d initiated=%d successful=%d",
					int(row[0]), int(row[1]), int(row[2]), int(row[3]), int(row[4]), int(row[5]))
			})
		} else {
			currentStat = stats.NewFile(configValues["StatsFile"])
		}

		var durationStat *stats.StatsFile
		if measureConnections {
			durationStat = stats.NewFile(configValues["MeasureConnections"])
		}

		for {
			time.Sleep(time.Second*5)
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
	defer stats.FlushAll()

	log.Print("Press enter to stop...\n\n")
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadLine()

}
