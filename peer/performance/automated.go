package performance

import (
	"sync"

	"go.dedis.ch/cs438/datastructures/concurrent"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

var LIKE = true
var DISLIKE = false

type AutomatedNode interface {
	GetReactions() map[string]map[string]bool
}

// Define a node that automatically reacts to publications
type PublicationReactorNode struct {
	// extends UserNode
	*impl.UserNode
	conf               *peer.Configuration
	reactToPublication func(BinaryPublication) error
	published          concurrent.Slice[BinaryPublication]
	reactionsLock      sync.Mutex
	reactions          map[string]map[string]bool
}

// Creates a PublicationReactorNode that react to publication with the
// provided with 'r'.
func NewPublicationReactorNode(conf peer.Configuration) *PublicationReactorNode {
	usernode := impl.NewUserNode(conf)
	r := func(BinaryPublication) error {
		return xerrors.Errorf("not implemented")
	}
	node := PublicationReactorNode{
		UserNode:           usernode,
		conf:               &conf,
		published:          concurrent.NewSlice[BinaryPublication](),
		reactToPublication: r,
		reactions:          make(map[string]map[string]bool),
	}
	conf.MessageRegistry.RegisterMessageCallback(types.BinaryPublicationMessage{}, node.handleBinaryPublicationMessage)
	return &node
}

// Handle BinaryPublicationMessage.
// 1. Process embedded PublicationMessage
// 2. React to the publication
func (n *PublicationReactorNode) handleBinaryPublicationMessage(msgType types.Message, pkt transport.Packet) error {
	msg := types.BinaryPublicationMessage{}
	err := n.conf.MessageRegistry.UnmarshalMessage(pkt.Msg, &msg)
	if err != nil {
		return xerrors.Errorf("error when handling binary publication msg: %v", err)
	}

	// pubMsg, err := n.conf.MessageRegistry.MarshalMessage(msg.Message)
	// if err != nil {
	// 	return xerrors.Errorf("error when handling binary publication msg: %v", err)
	// }

	// header := pkt.Header.Copy()
	// packet := transport.Packet{
	// 	Header: &header,
	// 	Msg:    &pubMsg,
	// }

	// // 1. Process embedded publication
	// err = n.conf.MessageRegistry.ProcessPacket(packet)
	// if err != nil {
	// 	return xerrors.Errorf("error when handling binary publication msg: %v", err)
	// }

	// 2. React to publication
	binaryPublication := binaryPublicationFromMessage(msg)
	err = n.reactToPublication(binaryPublication)
	if err != nil {
		return xerrors.Errorf("error when handling binary publication message: %v", err)
	}
	return nil
}

// Set the automated reaction to be the provided function
func (n *PublicationReactorNode) setReaction(r func(BinaryPublication) error) {
	n.reactToPublication = r
}

func (n *PublicationReactorNode) broadcastMessage(msg types.Message) error {
	marshalledMsg, err := n.conf.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return xerrors.Errorf("error when broadcasting msg: %v", err)
	}
	err = n.Broadcast(marshalledMsg)
	if err != nil {
		return xerrors.Errorf("error when broadcasting msg: %v", err)
	}
	return nil
}

func (n *PublicationReactorNode) Like(p peer.Publication) error {
	err := n.UserNode.Like(p)
	if err != nil {
		return err
	}

	n.addReaction(p.Author, p.ID, LIKE)
	return nil
}

func (n *PublicationReactorNode) Dislike(p peer.Publication) error {
	err := n.UserNode.Dislike(p)
	if err != nil {
		return err
	}
	n.addReaction(p.Author, p.ID, DISLIKE)
	return nil
}

func (n *PublicationReactorNode) addReaction(author string, ID string, value bool) {
	n.reactionsLock.Lock()
	defer n.reactionsLock.Unlock()
	r, ok := n.reactions[author]
	if !ok {
		newMap := make(map[string]bool)
		n.reactions[author] = newMap
		r = newMap
	}
	r[ID] = value
}

func (n *PublicationReactorNode) GetReactions() map[string]map[string]bool {
	n.reactionsLock.Lock()
	defer n.reactionsLock.Unlock()
	res := make(map[string]map[string]bool)
	for author, val := range n.reactions {
		inMap := make(map[string]bool)
		for id, v := range val {
			inMap[id] = v
		}
		res[author] = inMap
	}
	return res
}

func (n *PublicationReactorNode) GetAddress() string {
	return n.conf.Socket.GetAddress()
}

func (n *MaliciousNode) GetPublished() []BinaryPublication {
	return n.published.Elements()
}

func (n *MaliciousNode) Publish(p BinaryPublication) error {
	msg := binaryPublicationToMessage(p)
	err := n.broadcastMessage(msg)
	if err != nil {
		return xerrors.Errorf("error when publishing: %v", err)
	}
	n.published.Append(p)
	return nil
}

func (n *PublicationReactorNode) GetName() string {
	return n.conf.Socket.GetAddress()
}
