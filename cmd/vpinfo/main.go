package main

import (
	"fmt"
	"ipfs-connect2all/input"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please provide file as arg")
		return
	}
	peers, err := input.LoadVisitedPeers(os.Args[1])

	if err != nil {
		fmt.Printf("LVP error: %s\n", err)
	}

	limit := 10

	if peers != nil {
		fmt.Printf("Total number of peers: %d\n", len(peers))

		for pID, pAI := range peers {
			fmt.Printf("ID: %s\nID in AI:%s\n", pID.String(), pAI.NodeID.String())
			for i, addr := range pAI.MultiAddrs {
				fmt.Printf("MA%d: %s\n", i, addr.String())
			}
			fmt.Println()

			limit--
			if limit <= 0 {
				break
			}
		}
	}

}