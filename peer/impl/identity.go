package impl

import "go.dedis.ch/cs438/types"

// init consensus on identity
func (n *node) InitIdentityCheck(name, email, phone string) error {
	return n.Consensus(name, "1", types.Identity)
}

func (n *node) GetVerifiedUsers() []string {
	panic("not implemented!")
}

func (n *node) GetName() string {
	panic("not implemented!")
}
