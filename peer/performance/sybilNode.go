package performance

import (
	"errors"

	"go.dedis.ch/cs438/peer"
	"golang.org/x/xerrors"
)

type SybilNode struct {
	*PublicationReactorNode
	parent string
}

func NewSybilNode(conf peer.Configuration, parent string) *SybilNode {
	reactorNode := NewPublicationReactorNode(conf)
	node := SybilNode{
		PublicationReactorNode: reactorNode,
		parent:                 parent,
	}
	node.setReaction(node.reactToPublication)
	return &node
}

func (n *SybilNode) Start() error {
	if n.parent == "undefined" {
		return errors.New("error when starting sybil node: parent is missing")
	}
	return n.PublicationReactorNode.Start()
}

func (n *SybilNode) FollowParent() error {
	err := n.Follow(n.parent)
	return err
}

// Automatically react to the provided reaction:
//
//	if the author is the parent, like the publication
//	otherwise dislike the publication
func (n *SybilNode) reactToPublication(p BinaryPublication) error {
	var err error
	if n.isParent(p.Author) {
		err = n.Like(p.Publication)
	} else {
		err = n.Dislike(p.Publication)
	}
	if err != nil {
		return xerrors.Errorf("error when reacting to publication: %v", err)
	}
	return nil
}

func (n *SybilNode) isParent(user string) bool {
	return user == n.parent
}
