package impl

import (
	"fmt"

	"go.dedis.ch/cs438/types"
)

// init consensus on identity
func (n *node) InitIdentityCheck(name, email, phone string) {
	fmt.Println("IDENTITY CONSENSUS")
	n.Consensus(name, "1", types.Identity)
}

func (n *node) GetVerifiedUsers() []string {
	panic("not implemented!")
}

func (n *node) GetName() string {
	panic("not implemented!")
}
