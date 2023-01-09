package impl

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.dedis.ch/cs438/datastructures/concurrent"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

type RouteNode struct {
	*UserNode
	Profil *peer.SocialProfil
}

type RouteManager struct {
	RoutingTable *SybilRoutingTable
	// Stores the public key of a suspect at the tail
	KeyStore concurrent.Map[string, string]
	// Stores the tail of a route by id.
	TailStore           concurrent.Map[uint, types.Edge]
	NotificationService notificationService
}

func NewRouteManager(routeSeed int64, socialProfile peer.SocialProfil) *RouteManager {
	return &RouteManager{
		RoutingTable:        NewSybilRoutingTable(routeSeed, socialProfile),
		KeyStore:            concurrent.NewMap[string, string](),
		TailStore:           concurrent.NewMap[uint, types.Edge](),
		NotificationService: newNotificationService(),
	}
}

// Create a new RouteNode with an provided social network.
func NewRouteNode(user *UserNode) *RouteNode {
	node := &RouteNode{
		UserNode: user,
		Profil:   &user.routeManager.RoutingTable.socialProfil,
	}

	return node
}

// Executed by each suspect:
//
// 1. Select a random neighbor y.
//
// 2. Sends a ForwardRouteMessage to y.
//
// This function is blocking. It waits for the route to be completed.
//
// Each route need to have an id. This id is used to identify the route (to differentiate the routing permutations).
// It must be unique for each route, and in the range [1, r] for the verifier (as there is a mapping with the balance).
func (n *UserNode) StartRandomRoute(budget uint, id uint, timeout time.Duration, mustRegister bool) (string, error) {
	neighbor, ok := n.routeManager.RoutingTable.GetRandomNeighbor()
	if !ok {
		return "", xerrors.Errorf("StartRandomRoute{id: %v}: No neighbor to start random route", id)
	}

	key := n.getKey()
	err := n.sendRoute(budget, budget, key, id, neighbor, false, mustRegister)
	if err != nil {
		return "", xerrors.Errorf("StartRandomRoute{id: %v}: %v", id, err)
	}

	// Wait for the route to be completed
	tail, ok := n.routeManager.NotificationService.Wait(id, timeout)
	if !ok {
		return "", xerrors.Errorf("StartRandomRoute{id: %v}: timeout reached", id)
	}
	return tail, nil
}

// Build the route and sends it.
func (n *UserNode) sendRoute(routeLength uint, budget uint,
	key string, id uint, dest string, reversed bool, mustRegister bool) error {

	mac, err := n.computeMAC(budget, key, id, dest)
	if err != nil {
		return xerrors.Errorf("failed to compute MAC: %v", err)
	}
	routeMessage := types.RouteMessage{
		Length:       routeLength,
		Budget:       budget,
		Data:         key,
		MAC:          mac,
		ID:           id,
		Reversed:     reversed,
		MustRegister: mustRegister,
	}

	msg, err := n.conf.MessageRegistry.MarshalMessage(routeMessage)
	if err != nil {
		return xerrors.Errorf("Failed to marshal message: %v", err)
	}

	err = n.Unicast(dest, msg)
	if err != nil {
		return xerrors.Errorf("Failed to unicast: %v", err)
	}
	return nil
}

// Return the node public key.
func (n *UserNode) getKey() string {
	return n.conf.Socket.GetAddress()
}

// Compute a basic MAC to conform to the paper.
// The MAC is computed as follows:
// MAC = SHA256(Budget || Key || ID || SharedKey)
func (n *UserNode) computeMAC(budget uint, key string, id uint, dest string) (string, error) {
	sharedKey, ok := n.routeManager.RoutingTable.GetSharedKey(dest)
	if !ok {
		return "", xerrors.Errorf("computeMAC: Cannot compute the MAC without a shared key with %v", dest)
	}
	h := crypto.SHA256.New()
	h.Write([]byte(fmt.Sprint(budget)))
	h.Write([]byte(key))
	h.Write([]byte(fmt.Sprint(id)))
	h.Write([]byte(sharedKey))
	hash := h.Sum(nil)
	mac := hex.EncodeToString(hash)
	return mac, nil
}

const MIN_BUDGET uint = 1

func (n *UserNode) PropagateRandomRoute(msg types.RouteMessage, sender string) error {
	mac := msg.MAC
	budget := msg.Budget
	data := msg.Data
	id := msg.ID
	routeLength := msg.Length
	expectedMAC, err := n.computeMAC(budget, data, id, sender)
	if err != nil {
		return xerrors.Errorf("PropagateRandomRoute failed to compute MAC: %v", err)
	}
	if budget < 1 || budget > routeLength || mac != expectedMAC {
		return nil // Discard the message
	}

	if budget == MIN_BUDGET {
		// The message has reached its destination
		// If the message was a forward route from a s-instance, we need to register the public key
		// under the name of the edge.
		if !msg.Reversed {

			edge := types.Edge{From: sender, To: n.getKey()}
			edgeData := edge.String()
			if msg.MustRegister {
				publicKey := data
				n.routeManager.KeyStore.Add(edgeData, publicKey)
			}

			err := n.sendRoute(routeLength, routeLength, edgeData, id, sender, true, true)
			if err != nil {
				return xerrors.Errorf("PropagateRandomRoute: failed to reply with a reversed route: %v", err)
			}
		} else {
			tail, err := types.ParseEdge(data)
			if err != nil {
				return xerrors.Errorf("PropagateRandomRoute: the data {%v} received should be an edge", err)
			}
			n.routeManager.TailStore.Add(msg.ID, tail)
		}

	}
	nextHop, err := n.nextHop(msg, sender)
	if err != nil {
		err = n.sendRoute(routeLength, budget-1, data, id, nextHop, msg.Reversed, msg.MustRegister)
		if err != nil {
			return xerrors.Errorf("PropagateRandomRoute: %v", err)
		}
	}
	return nil
}

func (n *UserNode) nextHop(msg types.RouteMessage, sender string) (string, error) {
	id := msg.ID
	reversed := msg.Reversed
	hop, err := n.routeManager.RoutingTable.GetNextHop(id, sender, reversed)
	if err != nil {
		return "", xerrors.Errorf("nextHop: %v", err)
	}
	return hop, err
}

// A routing table to perform random routes.
type SybilRoutingTable struct {
	lock         sync.RWMutex
	socialProfil peer.SocialProfil
	// A source to generate the permutations.
	permutationSource *rand.Rand
	// The routing tables of the node. It corresponds to the permutations for each s-instance.
	// A route message coming from neighbor i is forwarded to the neighbor at index routingPermutations[id][i].
	routingPermutations map[uint][]uint
	// The reverse routing permutations. If x -> perm(x) in routingPermutations,
	// then perm(x) -> x in reversedRoutingPermutations.
	reversedRoutingPermutations map[uint][]uint

	suspectRouteID  uint
	verifierRouteID uint
}

func NewSybilRoutingTable(routeSeed int64, socialProfil peer.SocialProfil) *SybilRoutingTable {
	return &SybilRoutingTable{
		socialProfil:                socialProfil,
		routingPermutations:         make(map[uint][]uint),
		permutationSource:           rand.New(rand.NewSource(routeSeed)),
		reversedRoutingPermutations: make(map[uint][]uint),
	}
}

func (s *SybilRoutingTable) GetSharedKey(neighbor string) (string, bool) {
	return s.socialProfil.GetSharedKey(neighbor)
}

// Returns a random neighbor and true if any, or false otherwise.
func (s *SybilRoutingTable) GetRandomNeighbor() (string, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	size := s.socialProfil.NumberOfRelation()
	if size <= 0 {
		return "", false
	}
	index := (uint)(rand.Intn((int)(size)))
	return s.socialProfil.GetRelation(index), true
}

// Add a given permutation for the s-instance with the provided id.
// It overwrite the potential present permutation.
func (s *SybilRoutingTable) addRoutingPermutation(id uint, reversed bool) []uint {
	s.lock.Lock()
	defer s.lock.Unlock()
	nbrNeighbors := s.socialProfil.NumberOfRelation()
	perm := s.permutationSource.Perm((int)(nbrNeighbors))
	permCopy := make([]uint, len(perm))
	for i, p := range perm {
		permCopy[i] = uint(p)
	}
	currentPerm, ok := s.routingPermutations[id]
	if ok {
		return currentPerm
	}
	reversePerm := make([]uint, len(perm))
	for i, v := range perm {
		reversePerm[v] = uint(i)
	}
	s.routingPermutations[id] = permCopy
	s.reversedRoutingPermutations[id] = reversePerm
	if reversed {
		return reversePerm
	}
	return permCopy
}

// Returns the next hop for the route with the provided id, when the route comes from the provided sender.
// If reversed if true, the route uses the reverse permutation.
func (s *SybilRoutingTable) GetNextHop(id uint, sender string, reversed bool) (string, error) {
	readHop := func(id uint, sender string, reversed bool) (string, bool, error) {
		s.lock.RLock()
		defer s.lock.RUnlock()
		var perm []uint
		var ok bool
		if reversed {
			perm, ok = s.reversedRoutingPermutations[id]
		} else {
			perm, ok = s.routingPermutations[id]
		}

		if ok {
			hop, err := s.nextHopWithPerm(sender, perm)
			return hop, ok, err
		}

		return "", ok, nil
	}
	nextHop, isPresent, err := readHop(id, sender, reversed)
	if err != nil {
		return "", xerrors.Errorf("GetNextHop: %v", err)
	}

	if !isPresent {
		s.lock.Lock()
		defer s.lock.Unlock()
		perm := s.addRoutingPermutation(id, reversed)
		hop, err := s.nextHopWithPerm(sender, perm)
		if err != nil {
			return "", xerrors.Errorf("GetNextHop: %v", err)
		}
		return hop, nil
	}

	return nextHop, nil
}

func (s *SybilRoutingTable) nextHopWithPerm(neighbor string, perm []uint) (string, error) {
	neighborIndex, ok := s.socialProfil.GetRelationIndex(neighbor)
	if !ok {
		return "", fmt.Errorf("neighbor %s is not a neighbor", neighbor)
	}
	nextHopIndex := perm[neighborIndex]
	return s.socialProfil.GetRelation(nextHopIndex), nil
}

// A notification service to notify the completion of the route.
type notificationService struct {
	lock     sync.RWMutex
	channels map[uint]*chan string
}

func newNotificationService() notificationService {
	return notificationService{
		channels: make(map[uint]*chan string),
	}
}

// Wait for the notification of the route with the given id.
// Returns the tail of the route and true, if it succeeded within the timeout.
// Returns false otherwise.
// A timeout of 0 means that the function will wait indefinitely.
func (n *notificationService) Wait(id uint, timeout time.Duration) (string, bool) {
	timer := time.NewTimer(timeout)
	ch := make(chan string, 1)
	n.lock.Lock()
	n.channels[id] = &ch
	n.lock.Unlock()

	if timeout > 0 {
		select {
		case v, ok := <-ch:
			return v, ok
		case <-timer.C:
			// We ensure that no one can notify for this id anymore.
			n.lock.Lock()
			delete(n.channels, id)
			n.lock.Unlock()
			return "", false
		}
	} else {
		v, ok := <-ch
		return v, ok
	}

}

func (n *notificationService) Notify(id uint, data string) bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	ch, ok := n.channels[id]
	if !ok {
		return false
	}
	*ch <- data
	delete(n.channels, id)
	return true
}
