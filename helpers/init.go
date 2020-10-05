package helpers

import (
	"context"
	"fmt"
	"github.com/ipfs/go-bitswap/decision"
	"github.com/ipfs/go-bitswap/message"
	"github.com/ipfs/go-cid"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	fmt.Println("IPFS node created successfully! Peer ID: " + cfg.Identity.PeerID)

	return ipfs
}

func InitWantlistAnalysis(outfileDir string, snapshotInterval time.Duration, resetCache bool, dateFormat string,
	wantlistOfPeers map[peer.ID]bool) {
	decision.SetWantlistFilter(wantlistOfPeers)
	decision.EnableWantlistCaching(true)
	go func() {
		err := CheckOrCreateDir(outfileDir)
		if err != nil {
			return
		}

		for {
			time.Sleep(snapshotInterval)
			var wantLists map[peer.ID]map[cid.Cid]message.WantlistCacheEntry
			if resetCache {
				wantLists = decision.GetAndResetWantlistCache()
			} else {
				wantLists = decision.GetWantlistCache()
			}
			formattedDate := time.Now().Format(dateFormat)
			filename := outfileDir + "/wantlistLog_" + formattedDate + ".json"
			f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return
			}
			// TODO write json, problem: only strings as map types are allowed -> manually or write converter?
			var sb strings.Builder
			sb.WriteString("{\n")
			first0 := true
			for peerID, entryMap := range wantLists {
				if first0 {
					first0 = false
				} else {
					sb.WriteString(",\n")
				}
				sb.WriteByte('"')
				sb.WriteString(peerID.Pretty())
				sb.WriteString("\" : {\n")
				first1 := true
				for contentID, entry := range entryMap {
					if first1 {
						first1 = false
					} else {
						sb.WriteString(",\n")
					}
					sb.WriteByte('"')
					sb.WriteString(contentID.String())
					sb.WriteString("\" : { \"FirstWantHave\": \"")
					sb.WriteString(entry.FirstWantHave.String())
					sb.WriteString("\", \"LastWantHave\": \"")
					sb.WriteString(entry.LastWantHave.String())
					sb.WriteString("\", \"NumWantHave\": ")
					sb.WriteString(strconv.Itoa(entry.NumWantHave))
					sb.WriteString(", \"FirstWantBlock\": \"")
					sb.WriteString(entry.FirstWantBlock.String())
					sb.WriteString("\", \"LastWantBlock\": \"")
					sb.WriteString(entry.LastWantBlock.String())
					sb.WriteString("\", \"NumWantBlock\": ")
					sb.WriteString(strconv.Itoa(entry.NumWantBlock))
					sb.WriteString(" }")
				}
				sb.WriteString("\n}")
			}
			sb.WriteString("\n}")
			_, err = f.WriteString(sb.String())
			if err != nil {
				return
			}
			err = f.Sync()
			if err != nil {
				return
			}
			f.Close()
		}
	}()
}
