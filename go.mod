module ipfs-connect2all

require (
	github.com/ipfs/go-ipfs v0.5.1
	github.com/ipfs/go-ipfs-config v0.5.3
	github.com/ipfs/interface-go-ipfs-core v0.2.7
	github.com/libp2p/go-libp2p-core v0.5.3
	github.com/multiformats/go-multiaddr v0.2.1
	ipfs-crawler v0.0.0 //-20200603141538-ec2c9372e689
)

go 1.14

//replace ipfs-crawler => github.com/jatofg/ipfs-crawler master
replace ipfs-crawler => ../ipfs-crawler

//replace github.com/ipfs/go-ipfs => ../go-ipfs

//replace github.com/ipfs/go-bitswap => ../go-bitswap
