package input

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"ipfs-crawler/crawling"
	"strconv"
)

func CrawlDHT(configValues map[string]string, bootstrapPeers []*peer.AddrInfo) map[peer.ID]*peer.AddrInfo {

	crawlManagerConfig := crawling.ConfigureCrawlerManager()
	crawlManagerConfig.FilenameTimeFormat = configValues["DateFormat"]
	crawlManagerConfig.OutPath = configValues["DHTCrawlOut"] + "/"

	crawlWorkerConfig := crawling.Configure()
	crawlWorkerConfig.PreImagePath = configValues["DHTPreImages"]

	queueSize, err := strconv.Atoi(configValues["DHTQueueSize"])
	if err != nil {
		queueSize = 64384
	}
	cm := crawling.NewCrawlManagerV2WithConfig(queueSize, crawlManagerConfig)

	worker := crawling.NewIPFSWorkerWithConfig(0, context.Background(), crawlWorkerConfig)
	cm.AddWorker(worker)

	var bootstrapPeersWithCache []*peer.AddrInfo
	if configValues["DHTCacheFile"] != "" {
		cachedNodes, err := crawling.RestoreNodeCache(configValues["DHTCacheFile"])
		if err == nil {
			bootstrapPeersWithCache = make([]*peer.AddrInfo, len(bootstrapPeers), len(bootstrapPeers) + len(cachedNodes))
			copy(bootstrapPeersWithCache, bootstrapPeers)
			bootstrapPeersWithCache = append(bootstrapPeersWithCache, cachedNodes...)
		} else {
			bootstrapPeersWithCache = bootstrapPeers
		}
	} else {
		bootstrapPeersWithCache = bootstrapPeers
	}

	report := cm.CrawlNetwork(bootstrapPeersWithCache)
	startStamp := report.StartDate
	endStamp := report.EndDate
	crawling.ReportToFile(report,
		crawlManagerConfig.OutPath + fmt.Sprintf("visitedPeers_%s_%s.json", startStamp, endStamp))
	crawling.WritePeergraph(report,
		crawlManagerConfig.OutPath + fmt.Sprintf("peerGraph_%s_%s.csv", startStamp, endStamp))

	if configValues["DHTCacheFile"] != "" {
		crawling.SaveNodeCache(report, configValues["DHTCacheFile"])
	}

	ret := make(map[peer.ID]*peer.AddrInfo)
	for rID, rNode := range report.Nodes {
		addrInfo := &peer.AddrInfo{
			ID: rID,
			Addrs: rNode.MultiAddrs,
		}
		ret[rID] = addrInfo
	}

	return ret
}