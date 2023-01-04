package impl

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

type SybilLimitNodeImpl struct {
	peer.Peer
	conf                peer.Configuration
	routingTable        SybilRoutingTable
	keyStore            dataStore[string]
	tailStore           dataStore[uint]
	notificationService notificationService
}

func NewSybilLimitNodeImpl(conf peer.Configuration) *SybilLimitNodeImpl {
	node := &SybilLimitNodeImpl{
		Peer:                NewPeer(conf),
		conf:                conf,
		routingTable:        NewSybilRoutingTable(),
		keyStore:            newDataStore[string](),
		tailStore:           newDataStore[uint](),
		notificationService: newNotificationService(),
	}

	routeCallback := func(msg types.Message, pkt transport.Packet) error {
		v, ok := msg.(types.RouteMessage)
		if !ok {
			return xerrors.Errorf("%v is not a RouteMessage", msg)
		}
		return node.PropagateRandomRoute(v, pkt.Header.Source)
	}
	node.conf.MessageRegistry.RegisterMessageCallback(types.RouteMessage{}.NewEmpty(), routeCallback)

	return node
}

// Executed by each suspect:
// 1. Select a random neighbor y.
// 2. Sends a ForwardRouteMessage to y.

// This function is blocking. It waits for the route to be completed.

// Each route need to have an id. This id is used to identify the route (to differentiate the routing permutations).
// It must be unique for each route, and in the range [1, r] for the verifier (as there is a mapping with the balance).
func (n *SybilLimitNodeImpl) StartRandomRoute(budget uint, id uint, timeout time.Duration) (string, error) {
	neighbor, ok := n.routingTable.GetRandomNeighbor()
	if !ok {
		return "", xerrors.Errorf("No neighbor to start random route")
	}

	key := n.getKey()
	err := n.sendRoute(budget, budget, key, id, neighbor, false, true)
	if err != nil {
		return "", xerrors.Errorf("StartRandomRoute: %v", err)
	}

	// Wait for the route to be completed
	tail, ok := n.notificationService.Wait(id, timeout)
	if !ok {
		return "", xerrors.Errorf("StartRandomRoute: timeout reached")
	}
	return tail, nil
}

// Build the route and sends it.
func (n *SybilLimitNodeImpl) sendRoute(routeLength uint, budget uint,
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
func (n *SybilLimitNodeImpl) getKey() string {
	return n.conf.Socket.GetAddress()
}

// Compute a basic MAC to conform to the paper.
// The MAC is computed as follows:
// MAC = SHA256(Budget || Key || ID || SharedKey)
func (n *SybilLimitNodeImpl) computeMAC(budget uint, key string, id uint, dest string) (string, error) {
	sharedKey, err := n.routingTable.GetSharedKey(dest)
	if err != nil {
		return "", xerrors.Errorf("Cannot compute the MAC: %v", err)
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

func (n *SybilLimitNodeImpl) PropagateRandomRoute(msg types.RouteMessage, sender string) error {
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
			edge := toEdge(sender, n.getKey())
			if msg.MustRegister {
				publicKey := data
				n.keyStore.Register(edge, publicKey)
			}

			err := n.sendRoute(routeLength, routeLength, edge, id, sender, true, true)
			if err != nil {
				return xerrors.Errorf("PropagateRandomRoute: failed to reply with a reversed route: %v", err)
			}
		} else {
			tail := data
			n.tailStore.Register(msg.ID, tail)
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

func toEdge(src, dst string) string {
	return fmt.Sprintf("%s->%s", src, dst)
}

func (n *SybilLimitNodeImpl) nextHop(msg types.RouteMessage, sender string) (string, error) {
	id := msg.ID
	reversed := msg.Reversed
	if reversed {
		return n.routingTable.GetNextReversedHop(id, sender)
	}
	return n.routingTable.GetNextHop(id, sender)
}

// A routing table to perform random routes.
type SybilRoutingTable struct {
	// The social links of the node.
	socialNeighbors []string
	// A mapping from neighbors to their index.
	socialNeighborsMapping map[string]uint
	// The routing tables of the node. It corresponds to the permutations for each s-instance.
	// A route message coming from neighbor i is forwarded to the neighbor at index routingPermutations[id][i].
	routingPermutations map[uint][]uint
	// The reverse routing permutations. If x -> perm(x) in routingPermutations,
	// then perm(x) -> x in reversedRoutingPermutations.
	reversedRoutingPermutations map[uint][]uint

	// A key store for the key shared between the node and its neighbors
	keychain map[string]string
}

func NewSybilRoutingTable() SybilRoutingTable {
	return SybilRoutingTable{
		socialNeighbors:             []string{},
		socialNeighborsMapping:      make(map[string]uint),
		routingPermutations:         make(map[uint][]uint),
		reversedRoutingPermutations: make(map[uint][]uint),
		keychain:                    make(map[string]string),
	}
}

// Setup a shared key with the given neighbor.
// It is a dummy implementation to include a MAC.
func (s *SybilRoutingTable) setupKeyWith(neighbor string) {
	s.keychain[neighbor] = "DEFAULT_KEY"
}

func (s *SybilRoutingTable) GetSharedKey(neighbor string) (string, error) {
	key, ok := s.keychain[neighbor]
	if !ok {
		return "", xerrors.Errorf("No shared key with %s", neighbor)
	}
	return key, nil
}

// Add a social neighbor if it is not already a neighbor.
// Returns true if the neighbor was added, false otherwise.
func (s *SybilRoutingTable) AddSocialNeighbor(addr string) bool {
	_, isAlreadyNeighbor := s.socialNeighborsMapping[addr]
	if isAlreadyNeighbor {
		return false
	}
	index := uint(len(s.socialNeighbors))
	s.socialNeighbors = append(s.socialNeighbors, addr)
	s.socialNeighborsMapping[addr] = index
	s.setupKeyWith(addr)
	return true
}

// Returns a random neighbor and true if any, or false otherwise.
func (s *SybilRoutingTable) GetRandomNeighbor() (string, bool) {
	size := len(s.socialNeighbors)
	if size <= 0 {
		return "", false
	}
	index := rand.Intn(size)
	return s.socialNeighbors[index], true
}

// Add a given permutation for the s-instance with the provided id.
// If the permutation already exists, it is not replaced and it returns false.
func (s *SybilRoutingTable) AddRoutingPermutation(id uint, perm []uint) bool {
	permCopy := make([]uint, len(perm))
	copy(permCopy, perm)
	_, ok := s.routingPermutations[id]
	if ok {
		return false
	}
	reversePerm := make([]uint, len(perm))
	for i, v := range perm {
		reversePerm[v] = uint(i)
	}
	s.routingPermutations[id] = permCopy
	s.reversedRoutingPermutations[id] = reversePerm
	return true
}

func (s *SybilRoutingTable) GetNextHop(id uint, sender string) (string, error) {
	perm, ok := s.routingPermutations[id]
	if !ok {
		return "", fmt.Errorf("no permutation for id %d", id)
	}
	return s.nextHopWithPerm(sender, perm)
}

func (s *SybilRoutingTable) GetNextReversedHop(id uint, sender string) (string, error) {
	perm, ok := s.reversedRoutingPermutations[id]
	if !ok {
		return "", fmt.Errorf("no permutation for id %d", id)
	}
	return s.nextHopWithPerm(sender, perm)
}

func (s *SybilRoutingTable) nextHopWithPerm(neighbor string, perm []uint) (string, error) {
	neighborIndex, ok := s.socialNeighborsMapping[neighbor]
	if !ok {
		return "", fmt.Errorf("neighbor %s is not a neighbor", neighbor)
	}
	nextHopIndex := perm[neighborIndex]
	return s.socialNeighbors[nextHopIndex], nil
}

type dataStore[K comparable] struct {
	lock  sync.RWMutex
	store map[K]string
}

func newDataStore[K comparable]() dataStore[K] {
	return dataStore[K]{
		store: make(map[K]string),
	}
}

func (d *dataStore[K]) Register(key K, value string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.store[key] = value
}

func (d *dataStore[K]) Get(key K) (string, bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	value, ok := d.store[key]
	return value, ok
}

func (d *dataStore[K]) Delete(key K) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.store, key)
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
func (n *notificationService) Wait(id uint, timeout time.Duration) (string, bool) {
	timer := time.NewTimer(timeout)
	ch := make(chan string, 1)
	n.lock.Lock()
	n.channels[id] = &ch
	n.lock.Unlock()
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
