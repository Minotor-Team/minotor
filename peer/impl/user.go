package impl

import (
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/datastructures"
	"go.dedis.ch/cs438/datastructures/concurrent"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

type UserNode struct {
	peer.User
	peer.IdentityVerifier
	peer.SybilVerifier
	*PeersterNode
	followed concurrent.Set[string]
}

func NewUserNode(conf peer.Configuration) *UserNode {
	node := UserNode{
		PeersterNode: NewPeersterNode(conf),
		followed:     concurrent.NewSet[string](),
	}
	conf.MessageRegistry.RegisterMessageCallback(types.FollowRequest{}, node.handleFollowRequest)
	return &node
}

// implements peer.User
func (n *UserNode) Publish(content string) error {
	log.Info().Msg("Publish")
	return nil
}

// implements peer.User
func (n *UserNode) Like(p peer.Publication) error {
	log.Info().Msgf("%v: Like publication %v of %v", n.conf.Socket.GetAddress(), p.ID, p.Author)
	return nil
}

// implements peer.User
func (n *UserNode) Dislike(p peer.Publication) error {
	log.Info().Msgf("%v: Dislike publication %v of %v", n.conf.Socket.GetAddress(), p.ID, p.Author)
	return nil
}

// implements peer.User
func (n *UserNode) Follow(user string) error {
	// Dummy version of follow used for testing impl.
	// TO CHANGE
	msg := types.FollowRequest{
		RequestID: xid.New().String(),
		Source:    n.conf.Socket.GetAddress(),
	}

	marshalledMsg, err := n.reg.MarshalMessage(msg)
	if err != nil {
		return xerrors.Errorf("error when following: %v", err)
	}

	err = n.Unicast(user, marshalledMsg)
	if err != nil {
		return xerrors.Errorf("error when following: %v", err)
	}
	n.followed.Add(user)
	log.Info().Msgf("%v: Follow %v", n.conf.Socket.GetAddress(), user)
	return nil
}

// implements peer.User
func (n *UserNode) GetReputation() (int, error) {
	return 0, nil
}

// implements peer.User
func (n *UserNode) GetFollowed() datastructures.Set[string] {
	return n.followed.Values()
}

func (n *UserNode) GetIdentity() peer.Identity {
	return peer.Identity{}
}

// Define for testing implementation -- works with the testing Follow function
// Handle follow requests
// TO CHANGE OR REMOVE
func (n *UserNode) handleFollowRequest(msgType types.Message, pkt transport.Packet) error {
	msg := types.FollowRequest{}
	err := n.conf.MessageRegistry.UnmarshalMessage(pkt.Msg, &msg)
	if err != nil {
		return xerrors.Errorf("error when handling follow request: %v", err)
	}
	n.followed.Add(msg.Source)
	log.Info().Msgf("%v: Followback %v", n.conf.Socket.GetAddress(), msg.Source)
	return nil
}
