package impl

import (
	"fmt"

	"go.dedis.ch/cs438/types"
)

// init consensus on identity
func (n *node) InitIdentityCheck(name, email, phone string) {
	store := n.conf.Storage.GetIdentityStore()
	fmt.Printf("routing table start : %v\n", len(n.routingTable.getRoutingTable()))
	fmt.Printf("store start : %v\n", store.Len())
	fmt.Printf("TLC step start : %v\n", n.identityHandler.getStep())
	n.Consensus(name, "1", types.Identity)
	fmt.Printf("store end : %v\n", store.Len())
	fmt.Printf("TLC step end : %v\n", n.identityHandler.getStep())
	store.ForEach(func(key string, val []byte) bool {
		fmt.Printf("key - value = %v - %v\n", key, string(val))
		return true
	})
}

func (n *node) GetVerifiedUsers() []string {
	panic("not implemented!")
}

func (n *node) GetName() string {
	panic("not implemented!")
}
