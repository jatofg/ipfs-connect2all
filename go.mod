module ipfs-connect2all

require (
    github.com/ipfs/go-ipfs-config v0.2.0
    github.com/ipfs/go-ipfs-files v0.0.6
    github.com/ipfs/interface-go-ipfs-core v0.2.6
    github.com/libp2p/go-libp2p-peerstore v0.1.4
    github.com/ipfs/go-ipfs v0.5.0-dev
)

go 1.14

replace github.com/ipfs/go-ipfs => ../go-ipfs
replace github.com/ipfs/go-bitswap => ../go-bitswap
