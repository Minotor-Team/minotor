package unit

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Seed(t *testing.T) {
	rand.Seed(0)
	x1 := rand.Int()
	y1 := rand.Int()
	z1 := rand.Perm(10)
	rand.Seed(0)
	x2 := rand.Int()
	y2 := rand.Int()
	z2 := rand.Perm(10)
	require.Equal(t, x1, x2)
	require.Equal(t, y1, y2)
	require.Equal(t, z1, z2)
	t.Logf("x1: %d, y1: %d, z1: %v", x1, y1, z1)
}

func Test_Source_Seed(t *testing.T) {
	source := rand.NewSource(0)
	r1 := rand.New(source)
	x1 := r1.Int()
	x2 := r1.Int()

	source2 := rand.NewSource(0)
	rand.Seed(32)
	rand.Int()
	rand.Int()
	r2 := rand.New(source2)
	y1 := r2.Int()
	y2 := r2.Int()

	require.Equal(t, x1, y1)
	require.Equal(t, x2, y2)
}

func Test_For_Channels(t *testing.T) {
	ch := make(chan int, 3)
	go func() {
		ch <- 1
		time.Sleep(1 * time.Second)
		ch <- 2
		time.Sleep(1 * time.Second)
		ch <- 3
	}()
	j := 0
	for i := range ch {
		t.Logf("i: %d", i)
		j++
		if j == 3 {
			break
		}
	}
}

type TestSocialProfil struct {
	lock         sync.RWMutex
	relations    []string
	indexMapping map[string]int
}

func NewTestSocialProfil() *TestSocialProfil {
	return &TestSocialProfil{
		relations:    []string{},
		indexMapping: make(map[string]int),
	}
}

func (p *TestSocialProfil) GetRelations() []string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.relations
}

func (p *TestSocialProfil) GetRelation(n uint) string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.relations[n]
}

func (p *TestSocialProfil) GetRelationIndex(relation string) (uint, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	index, ok := p.indexMapping[relation]
	return uint(index), ok
}

func (p *TestSocialProfil) NumberOfRelation() uint {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return uint(len(p.relations))
}

func (p *TestSocialProfil) GetSharedKey(relation string) (string, bool) {
	return "SHARED_KEY", true
}

// type TestRouteNode struct {
// 	*z.TestNode
// 	Profil peer.SocialProfil
// }

// func NewTestRouteNode(t require.TestingT, transp transport.Transport, addr string, opts ...z.Option) *TestRouteNode {
// 	node := z.NewTestNode(t, peerFac, transp, addr, opts...)
// 	userNode, v := node.Peer.(*impl.UserNode)
// 	if !v {
// 		panic("Cannot create a routeNode from a non-userNode")
// 	}
// 	routeNode := impl.NewRouteNode(userNode)

// 	return &TestRouteNode{
// 		TestNode:     &node,
// 		RouteManager: routeNode.RouteManager,
// 	}
// }

// func (n *TestRouteNode) AddRelation() {

// }

// func Test_RouteNode_Route_Has_Correct_Length(t *testing.T) {
// 	nodes := createFullyConnectedNetwork(t, 2, z.WithAutostart(true))

// 	defer stopNodes(nodes)

// 	node1 := nodes[0]
// 	node2 := nodes[1]

// 	// A <-> B
// 	node1.Profil.relations = append(node1.Profil.relations, node2.Socket.GetAddress())
// 	node2.Profil.relations = append(node2.Profil.relations, node1.Socket.GetAddress())
// 	budget := uint(3)
// 	id := uint(0)

// 	time.Sleep(2 * time.Second)

// 	msg := types.ChatMessage{
// 		Message: "hello",
// 	}
// 	data, err := json.Marshal(&msg)
// 	require.NoError(t, err)

// 	tr := transport.Message{
// 		Type:    msg.Name(),
// 		Payload: data,
// 	}

// 	err = node1.Unicast(node2.Socket.GetAddress(), tr)
// 	require.NoError(t, err)
// 	tail, err := node1.StartRandomRoute(budget, id, 2*time.Second)
// 	require.NoError(t, err)
// 	t.Logf("Tail: %s", tail)
// 	// If everything goes well, the route should have a length of 2.
// }

// // Creates a fully connected network with no social links.
// func createFullyConnectedNetwork(t *testing.T, size uint, opts ...z.Option) []*TestRouteNode {
// 	nodes := make([]*TestRouteNode, size)
// 	opts = append(opts, z.WithTotalPeers(size))
// 	for i := uint(0); i < size; i++ {
// 		nodes[i] = NewTestChannelRouteNode(t, 0, localAddress("0"), opts...)
// 		nodes[i].Start()
// 		t.Logf("Node %d with address %s created. \n", i, nodes[i].Socket.GetAddress())
// 	}
// 	for i := uint(0); i < size; i++ {
// 		for j := uint(0); j < size; j++ {
// 			if i != j {
// 				nodes[i].AddPeer(nodes[j].Socket.GetAddress())
// 			}
// 		}
// 	}
// 	return nodes
// }

// type DetailledPacket struct {
// 	Packet transport.Packet
// 	msg    types.Message
// }

// func newDetailledPacket(t *testing.T, p transport.Packet, msg types.Message) DetailledPacket {
// 	err := json.Unmarshal(p.Msg.Payload, &msg)
// 	require.NoError(t, err)
// 	return DetailledPacket{
// 		Packet: p,
// 		msg:    msg,
// 	}
// }

// // func filterRouteMessage([] transport.Packet) []

// func stopNodes(nodes []*TestRouteNode) {
// 	for _, node := range nodes {
// 		node.Stop()
// 	}
// }

// func localAddress(port string) string {
// 	return fmt.Sprintf("127.0.0.0:%s", port)
// }
