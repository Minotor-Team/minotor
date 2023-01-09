package performance

import (
	"errors"
	"fmt"
	"sync"

	"go.dedis.ch/cs438/peer"
	"golang.org/x/xerrors"
)

type MaliciousNode struct {
	*PublicationReactorNode
	stateLock            sync.Mutex
	publications         []BinaryPublication
	children             []string
	publicationIDCounter int
}

func NewMaliciousNode(conf peer.Configuration) *MaliciousNode {
	reactorNode := NewPublicationReactorNode(conf)
	node := MaliciousNode{
		PublicationReactorNode: reactorNode,
		publications:           make([]BinaryPublication, 0),
		children:               make([]string, 0),
		publicationIDCounter:   0,
	}
	node.setReaction(node.reactToPublication)
	return &node
}

// Automatically react to publications. Only like its own publications.
func (n *MaliciousNode) reactToPublication(p BinaryPublication) error {
	var err error
	if p.Author == n.GetName() {
		err = n.Like(p.Publication)
	} else {
		err = n.Dislike(p.Publication)
	}
	if err != nil {
		return xerrors.Errorf("error when reacting to publication: %v", err)
	}
	return nil
}

// Publish all its publications.
func (n *MaliciousNode) Attack() error {
	n.stateLock.Lock()
	publications := make([]BinaryPublication, len(n.publications))
	copy(publications, n.publications)
	n.stateLock.Unlock()

	for _, p := range publications {
		err := n.Publish(p)
		if err != nil {
			return xerrors.Errorf("error when attacking: %v", err)
		}
	}
	return nil
}

func (n *MaliciousNode) AppendPublications(contents []string, truthValues []bool) error {
	if len(contents) != len(truthValues) {
		return errors.New("error: contents and truthvalues should have the same length")
	}
	publications := make([]BinaryPublication, 0)
	for idx, content := range contents {
		p := peer.Publication{
			ID:      fmt.Sprintf("%d", n.publicationIDCounter),
			Author:  n.GetName(),
			Content: content,
		}
		bp := BinaryPublication{
			Publication: p,
			TruthValue:  truthValues[idx],
		}
		n.publicationIDCounter++
		publications = append(publications, bp)
	}
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.publications = append(n.publications, publications...)
	return nil
}

func (n *MaliciousNode) AppendChildren(children ...string) {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	n.children = append(n.children, children...)
}
