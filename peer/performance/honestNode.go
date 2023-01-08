package performance

import (
	"go.dedis.ch/cs438/peer"
	"golang.org/x/xerrors"
)

// Define a honest node that automatically likes "true"
// content and dislikes "fake" content.
type HonestNode struct {
	*PublicationReactorNode
}

// Creates a new honest node
func NewHonestNode(conf peer.Configuration) *HonestNode {
	reactorNode := NewPublicationReactorNode(conf)
	node := HonestNode{
		PublicationReactorNode: reactorNode,
	}
	node.setReaction(node.reactToPublication)
	return &node
}

// React "honestly" to a binary publication (if the publication
// has fake content then dislike it otherwise like it)
func (n *HonestNode) reactToPublication(p BinaryPublication) error {
	var err error
	if p.isFake() {
		err = n.Dislike(p.Publication)
	} else {
		err = n.Like(p.Publication)
	}
	if err != nil {
		return xerrors.Errorf("error when reacting to publication: %v", err)
	}
	return nil
}
