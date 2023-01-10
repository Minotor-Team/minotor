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

	socialProfil peer.SocialProfil

	// SybilLimit protocol fields
	routeManager    *RouteManager
	suspectRouteID  rangedIndex
	verifierRouteID rangedIndex

	// Tails sent by suspect waiting to be verified.
	pendingSuspectTails chan pendingSuspect
	processedSuspects   concurrent.Set[string]
	verifier            Verifier
}

// Creates a new UserNode with a provided state.
// To initialize a UserNode with a default state, use NewUserNode.
func NewUserNode(conf peer.Configuration) *UserNode {
	node := UserNode{
		PeersterNode:    NewPeersterNode(conf),
		followed:        concurrent.NewSet[string](),
		routeManager:    NewRouteManager(conf.RouteSeed, peer.NewSocialProfilAdapter()),
		suspectRouteID:  rangedIndex{low: conf.NumberRoutes, high: 2 * conf.NumberRoutes},
		verifierRouteID: rangedIndex{low: 0, high: conf.NumberRoutes},
	}
	conf.MessageRegistry.RegisterMessageCallback(types.FollowRequest{}, node.handleFollowRequest)

	routeCallback := func(msg types.Message, pkt transport.Packet) error {
		v, ok := msg.(types.RouteMessage)
		if !ok {
			return xerrors.Errorf("%v is not a RouteMessage", msg)
		}
		return node.PropagateRandomRoute(v, pkt.Header.Source)
	}
	node.conf.MessageRegistry.RegisterMessageCallback(types.RouteMessage{}.NewEmpty(), routeCallback)
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

type Tails = map[uint]types.Edge

type Verifier struct {
	tails map[uint]types.Edge
	// A data structure to compute instersections efficiently.
	// It maps an edge to the list of v-instance IDs that have this edge as a tail.
	tailsSet map[types.Edge][]uint

	// A array of indices [0, r - 1] for the loads of the balance condition.
	verifierBalanceLoads []uint
	// Sum of the loads, i.e., (1 + sum(verifierBalanceLoads)) / r
	totalLoads float64

	// Corresponds to the parameter b in the paper.
	// Dynamic upper bound to the number of sybil node accepted by attack edge.
	sybilBound float64

	// A notification service for registration query
	notificationService peer.NotificationService[uint, bool]

	// A set of rejected suspects
	rejectedSuspects datastructures.Set[string]
}

func NewVerifier(routeLength uint) *Verifier {
	return &Verifier{
		tails:                make(map[uint]types.Edge),
		tailsSet:             make(map[types.Edge][]uint),
		verifierBalanceLoads: make([]uint, routeLength),
		notificationService:  NewNotificationService[uint, bool](),
	}
}

func (v *Verifier) AddTails(tails Tails) {
	v.tails = tails
	v.tailsSet = reverse(tails)
}

// Compute the intersection between the tails of the v-instances and the provided tails.
// It returns a map from v-instance number to tail.
func (v *Verifier) Intersection(tails Tails) Tails {
	intersection := make(Tails)
	for _, tail := range tails {
		ids, ok := v.tailsSet[tail]
		if ok {
			addAll(intersection, tail, ids)
		}

		rev := tail.Reverse()
		ids, ok = v.tailsSet[rev]
		if ok {
			addAll(intersection, rev, ids)
		}
	}
	return intersection

}

func (v *Verifier) Reject(suspect string) {
	v.rejectedSuspects.Add(suspect)
}

// Returns true if the value is below the sybil upper bound.
func (v *Verifier) SatisfiesSybilBound(value uint) bool {
	bound := uint(v.sybilBound)
	return !(value > bound)
}

// Returns the instance number of the v-instance with the smallest load together with the value.
// The output is of the form (instanceNumber, load)
// Returns false if instanceNumbers is empty.
func (v *Verifier) MinimunLoadInstanceNumber(instanceNumbers []uint) (uint, uint, bool) {
	if len(instanceNumbers) == 0 {
		return 0, 0, false
	}
	minLoad := uint(0)
	minInstanceNumber := uint(0)
	for _, instanceNumber := range instanceNumbers {
		load := v.verifierBalanceLoads[instanceNumber]
		if load < minLoad {
			minLoad = load
			minInstanceNumber = instanceNumber
		}
	}

	return minInstanceNumber, minLoad, true
}

func (v *Verifier) IncrementLoad(instanceNumber uint) {
	v.verifierBalanceLoads[instanceNumber] += 1
	v.totalLoads += 1
}

func addAll(m Tails, tail types.Edge, values []uint) {
	for _, v := range values {
		m[v] = tail
	}
}

func (v *Verifier) GetTailSet() map[types.Edge][]uint {
	res := make(map[types.Edge][]uint)
	for key, value := range v.tailsSet {
		valueCopy := make([]uint, len(value))
		copy(valueCopy, value)
		res[key] = valueCopy
	}

	return res
}

// Compute the inverse map. If A -> B and C -> B, then B -> [A, C].
func reverse(tails Tails) map[types.Edge][]uint {
	res := make(map[types.Edge][]uint)
	for k, v := range tails {
		idArray, ok := res[v]
		if !ok {
			idArray = make([]uint, 0)
		}

		idArray = append(idArray, k)
		res[v] = idArray
	}

	return res
}
