# ipfs-connect2all

An academic tool which attempts to connect to all peers discovered in the IPFS network.

* Build `cmd/ipfs-connect2all/main.go` for the `ipfs_connect2all` tool
* Build `cmd/c2a_analysis/main.go` for the `c2a_analysis` tool
* Build `cmd/c2a_analyzeall/main.go` for the `c2a_analyzeall` tool

## ipfs_connect2all

The main tool. Runs a `go-ipfs` daemon and attempts to connect to all peers it gets to know. It records 
statistical data (see output files) and can optionally write snapshots of peer lists in certain intervals. 
It can also initiate DHT crawls using [ipfs-crawler](https://github.com/wiberlin/ipfs-crawler) and/or use 
the result of these crawls to connect to the peers found.

**Usage and options:**

```
Usage: ipfs_connect2all [options]

General options:
Help                      Show this help message and quit
LogToStdout               Write stats to stdout
StatsFile=<file>          Write to stats file <file> (default: peersStat.dat)
StatsInterval=<dur>       Stats collecting interval (default: 5s) 
                          (units available: ms, s, m, h)
ConnMgrType=basic         Use basic IPFS connection manager (instead of none)
ConnMgrHighWater=<value>  Max. number of peers in IPFS connection manager
                          (default: 0)
DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)
MeasureConnections=<file> Track average connection time and write to <file> 
                          (default: no tracking, reduces concurrency)

Snapshot options:
Snapshots=<dir>           Write snapshots of currently known/... peers to files
                          in <dir> (no trailing /)
SnapshotInterval=<dur>    Snapshot interval (default: 10m)

DHT scan options:
DHTPeers=<file>           Load visited peers from DHT crawl from 
                          visitedPeers*.json file <file>
DHTConnsPerSec=<value>    Initiate <value> connections to peers from DHT crawl
                          per second (default: 5)

ipfs-crawler integration options:
DHTCrawlInterval=<dur>    Crawl the DHT automatically in intervals of <dur>
                          (default: off)
DHTCrawlOut=<dir>         Directory for saving the output files of DHT crawls
                          (default: crawl)
DHTPreImages=<file>       File with preimages for DHT crawls
                          (default: precomputed_hashes/preimages.csv)
DHTQueueSize=<size>       Queue size for DHT crawls (default: 64384)
DHTCacheFile=<file>       Cache file for DHT crawls (default:
                          crawls/nodes.cache; empty to disable caching)
```

### Output files format

#### Stats file

Gnuplot-compatible DAT/TSV file (tab-separated values) containing one data point (row with numerical values) 
every `StatsInterval` seconds.

**Columns:**

1. Known peers in go-ipfs
1. Connected peers in go-ipfs
1. Established connections (manually initiated, still connected) by connect2all
1. Failed connections (manually initiated) by connect2all
1. Connections initiated by connect2all (but still pending, not yet established or failed)
1. Successful connections (once established) by connect2all (incl. lost connections)

#### Connection measurement file

**Columns:**

1. Total mean connection duration
1. Mean connection duration of successful connection attempts
1. Mean connection duration of failed connection attempts

#### Snapshot files

* `known_*`: List of known peers in go-ipfs at a certain point in time, one peer ID per line.
* `connected_*`: CSV file (comma-separated) of connected peers in go-ipfs at a certain point in time, 
  contains the peer ID in the first column, the direction of connection 
  in the second column, and the IPFS version (if available) in the third column.
* `established_*`: List of peers with manually established connections by connect2all, one peer ID per line.
* `failed_*`: List of peers with failed connection attempts by connect2all, one peer ID per line.
* `successful_*`: List of peers with a once successful connection (see above) by connect2all, one peer ID per line.

## c2a_analysis

Takes a timestamp and the directories of crawl output files and snapshots as arguments, compares the 
peers in the crawl and snapshot files closest after this timestamp, computes some statistical measures, 
and prints them.

**Usage:**

```
Usage: c2a_analysis [arguments]

Required arguments:
Timestamp=<value>         Timestamp to use (see DateFormat, next crawl and
                          snapshots from this point will be used)

Optional arguments:
DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)
DHTCrawlDir=<dir>         Directory in which the snapshots are located
SnapshotDir=<dir>         Directory in which the crawl output files are located
SnapshotTS=<value>        Timestamp to use for snapshots (if given, the one
                          from above will be used for the crawl only)
```

## c2a_analyzeall

Compares all crawl output files with snapshots at about the start time of the crawl, as well as 
with those taken 10, 20, and 30~minutes later. It can also automatically calculate total numbers of 
known and connected peers, established, successful, and failed connections from the snapshots, and 
analyze the churn between the snapshots. In addition, it prints some statistical data about the 
length of connections.

**Usage:**

```
Usage: c2a_analyzeall [options]

Options:
DateFormat=<format>       Date format (Go-style) (default: 06-01-02--15:04:05)
DHTCrawlDir=<dir>         Directory in which the snapshots are located
                          (default: snapshots)
SnapshotDir=<dir>         Directory in which the crawl output files are located
                          (default: crawls)
OutputDir=<dir>           Directory to which the output files will be saved
                          (default: analysis_result)
SkipComparisons           Do not calculate comparisons
SkipChurn                 Do not calculate churn
SkipTotal                 Do not record total numbers
```

### Statistical output files

Numerical columns, using a Gnuplot-compatible DAT/TSV format.

#### Comparison files
comparison_0m.dat, comparison_10m.dat, comparison_20m.dat, comparison_30m.dat

Comparison of the peers found in the DHT crawl to those known to ipfs-connect2all at about the same time (0m), resp. 
10/20/30 minutes later. One data point (line) for each crawl.

**Columns:**

1. Peers found in DHT crawl
1. Reachable peers found in DHT crawl
1. Known peers in go-ipfs
1. Connected peers in go-ipfs
1. Successful connections by connect2all
1. Failed connections by connect2all
1. Peers marked reachable in crawl, but not known by go-ipfs
1. Peers marked reachable in crawl, but not connected in go-ipfs
1. Peers marked reachable in crawl, but without successful connection by connect2all
1. Peers marked reachable in crawl, but failed in connect2all
1. Peers known by go-ipfs, but not in crawl
1. Peers connected to go-ipfs, but not in crawl
1. Peers connected to go-ipfs, but not marked reachable in crawl
1. Peers with successful connection by connect2all, but not in crawl
1. Peers with successful connection by connect2all, but not marked reachable in crawl

#### Snapshot totals file
total.dat

Contains the total numbers of peers. One data point (line) for each snapshot.

**Columns:**

1. Known peers in go-ipfs
1. Connected peers in go-ipfs
1. Established connections by connect2all
1. Failed connections by connect2all
1. Successful connections by connect2all

#### Snapshot churn file
churn.dat

Contains the churn of peers/connections in the snapshots. One data point (line) 
for each snapshot.

**Columns:**

1. Newly known peers (since the last snapshot)
1. New connections (since the last snapshot)
1. Lost connections (since the last snapshot)

## Scripts

(in the `scripts` directory)

### datstats.py

Takes a DAT/TSV file and computes the minimum, maximum, mean, and average of each column.

### analysis.gnuplot

Example gnuplot file for plotting the output files of `c2a_analyzeall`, assuming a multiple-days run. 
It might be necessary to comment out all but one `plot` statements to get the desired plot.
