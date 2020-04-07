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
	"ipfs-connect2all/stats"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

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
	// disable connection management
	cfg.Swarm.ConnMgr.Type = "none"

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
		connectionsMutex.Unlock()
	}

	countConnections := func() (int, int, int) {
		connectionsMutex.Lock()
		defer connectionsMutex.Unlock()
		return len(connectionsEstablished), len(connectionsFailed), len(connectionsInitiated)
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

	// function to attempt to connect to a node and track progress
	tryToConnect := func(peerInfo peer.AddrInfo) {
		if !checkConnectionAndSetInitiated(peerInfo.ID) {
			return
		}

		err := ipfs.Swarm().Connect(ctx, peerInfo)
		if err == nil {
			setConnectionEstablished(peerInfo.ID)
		} else {
			//log.Printf("Could not connect to peer: %s", err)
			setConnectionFailed(peerInfo.ID)
		}
	}

	// collect number of connected and known peers every 5s, try to connect to known peers
	// write stats to log file peersStat.dat
	go func() {
		currentStat := stats.NewFileWithCallback("peersStat.dat", func(row []float64) {
			log.Printf("known=%d connected=%d established=%d failed=%d initiated=%d",
				int(row[0]), int(row[1]), int(row[2]), int(row[3]), int(row[4]))
		})

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
			manEstablished, manFailed, manInitiated := countConnections()
			currentStat.AddInts(len(knownPeers), len(connectedPeers), manEstablished, manFailed, manInitiated)

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
					go tryToConnect(peer.AddrInfo{peerID, peerAddr})
				}
			}
		}
	}()

	// flush data to file every 30s and at the end
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
