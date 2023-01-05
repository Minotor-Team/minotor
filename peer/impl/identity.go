package impl

import (
	"fmt"

	"go.dedis.ch/cs438/types"
)

// init consensus on identity
func (n *node) InitIdentityCheck(name, email, phone string) {
	fmt.Println("INIT CONSENSUS")
	n.Consensus(name, "1", types.Identity)
}
