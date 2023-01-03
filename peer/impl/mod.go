package impl

import (
	"sync"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// creates a new peer
func NewPeer(conf peer.Configuration) peer.Peer {

	node := &node{
		wg:                sync.WaitGroup{},
		run:               uint32(0),
		stopChannel:       make(chan struct{}),
		conf:              conf,
		soc:               conf.Socket,
		reg:               conf.MessageRegistry,
		routingTable:      newNodeRT(),
		rumorsHandler:     newRumorsHandler(),
		acksHandler:       newChannelsHandler(),
		catalog:           newCatalog(),
		requestsHandler:   newChannelsHandler(),
		store:             newStore(),
		searchsHandler:    newChannelsHandler(),
		paxosHandler:      newPaxosHandler(conf),
		messageReputation: newMsgReputation(),
	}

	myAddr := node.soc.GetAddress()
	node.SetRoutingEntry(myAddr, myAddr)
	node.reg.RegisterMessageCallback(types.ChatMessage{}, node.ExecChatMessage)
	node.reg.RegisterMessageCallback(types.RumorsMessage{}, node.ExecRumorsMessage)
	node.reg.RegisterMessageCallback(types.EmptyMessage{}, node.ExecEmptyMessage)
	node.reg.RegisterMessageCallback(types.StatusMessage{}, node.ExecStatusMessage)
	node.reg.RegisterMessageCallback(types.AckMessage{}, node.ExecAckMessage)
	node.reg.RegisterMessageCallback(types.PrivateMessage{}, node.ExecPrivateMessage)
	node.reg.RegisterMessageCallback(types.DataReplyMessage{}, node.ExecDataReplyMessage)
	node.reg.RegisterMessageCallback(types.DataRequestMessage{}, node.ExecDataRequestMessage)
	node.reg.RegisterMessageCallback(types.SearchReplyMessage{}, node.ExecSearchReplyMessage)
	node.reg.RegisterMessageCallback(types.SearchRequestMessage{}, node.ExecSearchRequestMessage)
	node.reg.RegisterMessageCallback(types.PaxosPrepareMessage{}, node.ExecPaxosPrepareMessage)
	node.reg.RegisterMessageCallback(types.PaxosProposeMessage{}, node.ExecPaxosProposeMessage)
	node.reg.RegisterMessageCallback(types.PaxosPromiseMessage{}, node.ExecPaxosPromiseMessage)
	node.reg.RegisterMessageCallback(types.PaxosAcceptMessage{}, node.ExecPaxosAcceptMessage)
	node.reg.RegisterMessageCallback(types.TLCMessage{}, node.ExecTLCMessage)

	// reputation
	node.reg.RegisterMessageCallback(types.LikeMessage{}, node.ExecLikeMessage)
	node.reg.RegisterMessageCallback(types.DislikeMessage{}, node.ExecDislikeMessage)

	return node
}

// implements a peer to build a Peerster system
type node struct {
	wg              sync.WaitGroup
	run             uint32
	stopChannel     chan struct{}
	conf            peer.Configuration
	soc             transport.Socket
	reg             registry.Registry
	routingTable    *nodeRT
	rumorsHandler   *rumorsHandler
	acksHandler     *channelsHandler
	catalog         *catalog
	requestsHandler *channelsHandler
	store           *store
	searchsHandler  *channelsHandler
	paxosHandler    *paxosHandler
	// reputation !
	// reputation        *types.ReputationValue
	messageReputation *messageReputation
}

type void struct{}

var member void
