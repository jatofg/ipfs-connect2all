package helpers

import (
	"context"
	"fmt"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"io/ioutil"
	"path/filepath"
)

// spawn node on temporary repository
func InitIpfs(ctx context.Context, connMgrType string, connMgrHighWater int, portPrefix string) iface.CoreAPI {

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
	// use server profile to avoid problems
	_ = config.Profiles["server"].Transform(cfg)
	// custom config values
	cfg.Addresses.Swarm = []string{"/ip4/0.0.0.0/tcp/" + portPrefix + "4001",
		                           "/ip6/::/tcp/" + portPrefix + "4001",
		                           "/ip4/0.0.0.0/udp/" + portPrefix + "4001/quic",
		                           "/ip6/::/udp/" + portPrefix + "4001/quic"}
	cfg.Addresses.API = []string{"/ip4/127.0.0.1/tcp/" + portPrefix + "5001"}
	cfg.Addresses.Gateway = []string{"/ip4/127.0.0.1/tcp/" + portPrefix + "8080"}
	cfg.Swarm.ConnMgr.Type = connMgrType
	cfg.Swarm.ConnMgr.HighWater = connMgrHighWater

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