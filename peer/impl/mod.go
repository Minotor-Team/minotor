package impl

import (
	"go.dedis.ch/cs438/peer"
)

// creates a new peer
func NewPeer(conf peer.Configuration) peer.Peer {
	userNode := NewUserNode(conf)
	return userNode
}

type void struct{}

var member void
