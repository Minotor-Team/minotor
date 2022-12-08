package impl

import (
	"math/rand"
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
)

// initialises node routing table
func newNodeRT() *nodeRT {
	return &nodeRT{
		routingTable: make(peer.RoutingTable),
		neighbours:   []string{},
	}
}

type nodeRT struct {
	sync.RWMutex
	routingTable peer.RoutingTable
	neighbours   []string
}

// adds new known address to the routing table
func (nRT *nodeRT) addEntry(k, v string) {
	nRT.Lock()
	defer nRT.Unlock()

	nRT.routingTable[k] = v
}

// adds new known addresses to the routing table and neighbours
func (nRT *nodeRT) addEntries(addr []string) {
	nRT.Lock()
	defer nRT.Unlock()

	for _, k := range addr {
		nRT.routingTable[k] = k
		nRT.neighbours = append(nRT.neighbours, k)
	}
}

// gets addr in routing table from given key
func (nRT *nodeRT) getAddr(dest string) (string, bool) {
	nRT.RLock()
	defer nRT.RUnlock()

	addr, inTable := nRT.routingTable[dest]
	return addr, inTable
}

// deletes key from routing table
func (nRT *nodeRT) delete(key string) {
	nRT.Lock()
	defer nRT.Unlock()

	delete(nRT.routingTable, key)
}

// returns a copy of the node's routing table
func (nRT *nodeRT) getRoutingTable() peer.RoutingTable {
	nRT.RLock()
	defer nRT.RUnlock()

	// initialize table to be returned
	copyRT := make(peer.RoutingTable)

	// copy routing table
	for k, v := range nRT.routingTable {
		copyRT[k] = v
	}

	return nRT.routingTable
}

// returns all node's neighbours
func (nRT *nodeRT) getNeighbours() []string {
	nRT.RLock()
	defer nRT.RUnlock()

	// initialize table to be returned and copy neighbours
	copyN := make([]string, len(nRT.neighbours))
	copy(copyN, nRT.neighbours)

	// shuffle array to randomize neighbours choice
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(copyN), func(i, j int) { copyN[i], copyN[j] = copyN[j], copyN[i] })

	return copyN
}

// updates routing table if needed
func (nRT *nodeRT) updateRT(source string, relay string) {
	nRT.Lock()
	defer nRT.Unlock()

	// check if source is not a neighbour
	for _, n := range nRT.neighbours {
		if source == n {
			return
		}
	}

	// update node's neighbours if source is a neighbour
	if source == relay {
		nRT.neighbours = append(nRT.neighbours, source)
	}

	// add new known address to the node
	nRT.routingTable[source] = relay
}
