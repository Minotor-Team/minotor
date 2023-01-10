package impl

import (
	"go.dedis.ch/cs438/types"
)

// init consensus on identity
func (n *node) InitIdentityCheck(name, email, phone string) {
	n.Consensus(name, "1", types.Identity)
}
