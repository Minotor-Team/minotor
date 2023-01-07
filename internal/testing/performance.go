package testing

import (
	"math/rand"
	"sync"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/performance"
	"go.dedis.ch/cs438/transport"
)

type HonestTestNode struct {
	*performance.HonestNode
	config peer.Configuration
	socket transport.ClosableSocket
	t      require.TestingT
}

func buildConfig(t require.TestingT, trans transport.Transport,
	addr string, opts ...Option) (configTemplate, peer.Configuration, transport.ClosableSocket) {
	template := newConfigTemplate()
	for _, opt := range opts {
		opt(&template)
	}

	socket, err := trans.CreateSocket(addr)
	require.NoError(t, err)

	config := peer.Configuration{}

	config.Socket = socket
	config.MessageRegistry = template.registry
	config.AntiEntropyInterval = template.AntiEntropyInterval
	config.HeartbeatInterval = template.HeartbeatInterval
	config.ContinueMongering = template.ContinueMongering
	config.AckTimeout = template.AckTimeout
	config.Storage = template.storage
	config.ChunkSize = template.chunkSize
	config.BackoffDataRequest = template.dataRequestBackoff
	config.TotalPeers = template.totalPeers
	config.PaxosThreshold = template.paxosThreshold
	config.PaxosID = template.paxosID
	config.PaxosProposerRetry = template.paxosProposerRetry

	return template, config, socket
}

func NewHonestTestNode(t require.TestingT, trans transport.Transport,
	addr string, opts ...Option) HonestTestNode {
	template, config, socket := buildConfig(t, trans, addr, opts...)

	node := performance.NewHonestNode(config)

	require.Equal(t, len(template.messages), len(template.handlers))
	for i, msg := range template.messages {
		config.MessageRegistry.RegisterMessageCallback(msg, template.handlers[i])
	}

	if template.autoStart {
		err := node.Start()
		require.NoError(t, err)
	}

	return HonestTestNode{
		HonestNode: node,
		config:     config,
		socket:     socket,
		t:          t,
	}
}

func (m *HonestTestNode) GetAdress() string {
	return m.socket.GetAddress()
}

// GetIns returns all the messages received so far.
func (t HonestTestNode) GetIns() []transport.Packet {
	return t.socket.GetIns()
}

// GetOuts returns all the messages sent so far.
func (t HonestTestNode) GetOuts() []transport.Packet {
	return t.socket.GetOuts()
}

type MaliciousTestNode struct {
	*performance.MaliciousNode
	config peer.Configuration
	socket transport.ClosableSocket
	sybils []*SybilTestNode
	t      require.TestingT
}

func NewMaliciousTestNode(t require.TestingT, trans transport.Transport,
	addr string, opts ...Option) MaliciousTestNode {

	template, config, socket := buildConfig(t, trans, addr, opts...)

	node := performance.NewMaliciousNode(config)

	require.Equal(t, len(template.messages), len(template.handlers))
	for i, msg := range template.messages {
		config.MessageRegistry.RegisterMessageCallback(msg, template.handlers[i])
	}

	if template.autoStart {
		err := node.Start()
		require.NoError(t, err)
	}

	return MaliciousTestNode{
		MaliciousNode: node,
		config:        config,
		socket:        socket,
		sybils:        make([]*SybilTestNode, 0),
		t:             t,
	}
}

func (t *MaliciousTestNode) Stop() error {
	var err error
	for _, sybil := range t.sybils {
		err = sybil.Stop()
		if err != nil {
			return err
		}
	}
	return t.MaliciousNode.Stop()
}

// GetIns returns all the messages received so far.
func (t MaliciousTestNode) GetIns() []transport.Packet {
	return t.socket.GetIns()
}

// GetOuts returns all the messages sent so far.
func (t MaliciousTestNode) GetOuts() []transport.Packet {
	return t.socket.GetOuts()
}

// Creates sybil nodes for this malicious nodes. Sybil nodes are peers with their parent.
// Each Sybil node follow its parent (and the parent follow all sybil node).
// The Sybil nodes do not follow eachother.
func (n *MaliciousTestNode) CreateSybilNodes(nbr int, t require.TestingT, trans transport.Transport,
	addr string, opts ...Option) []*SybilTestNode {
	res := make([]*SybilTestNode, nbr)
	for i := 0; i < nbr; i++ {
		res[i] = n.createSybilNode(t, trans, addr, opts...)
	}
	n.sybils = append(n.sybils, res...)
	return res
}

func (n *MaliciousTestNode) createSybilNode(t require.TestingT, trans transport.Transport,
	addr string, opts ...Option) *SybilTestNode {
	res := NewSybilTestNode(t, trans, addr, n, opts...)
	err := res.FollowParent()
	require.NoError(t, err)
	return &res
}

// Array of pointers to all children
func (n *MaliciousTestNode) GetChildren() []*SybilTestNode {
	return n.sybils
}

type SybilTestNode struct {
	*performance.SybilNode
	config peer.Configuration
	socket transport.ClosableSocket
	t      require.TestingT
}

func NewSybilTestNode(t require.TestingT, trans transport.Transport,
	addr string, parent *MaliciousTestNode, opts ...Option) SybilTestNode {

	template, config, socket := buildConfig(t, trans, addr, opts...)

	node := performance.NewSybilNode(config, parent.GetName())
	// Make them peer of eachother
	node.AddPeer(parent.GetAddress())
	parent.AddPeer(node.GetAddress())

	require.Equal(t, len(template.messages), len(template.handlers))
	for i, msg := range template.messages {
		config.MessageRegistry.RegisterMessageCallback(msg, template.handlers[i])
	}

	if template.autoStart {
		err := node.Start()
		require.NoError(t, err)
	}

	return SybilTestNode{
		SybilNode: node,
		config:    config,
		socket:    socket,
		t:         t,
	}
}

// GetIns returns all the messages received so far.
func (t SybilTestNode) GetIns() []transport.Packet {
	return t.socket.GetIns()
}

// GetOuts returns all the messages sent so far.
func (t SybilTestNode) GetOuts() []transport.Packet {
	return t.socket.GetOuts()
}

// Create a cluster of honest nodes. A cluster of nodes
// is a connected graph where edges are network routes (ie.
// each node knows how to reach eachohter). Nodes do not
// necessarily follow each other.
type HonestNodeCluster struct {
	nodes          []*HonestTestNode
	maliciousNodes []*MaliciousTestNode
	t              require.TestingT
}

// Set up an entire cluster with 'nbrHonest' honest nodes and
// 'nbrMalicious' malicious nodes. Two distinct honest nodes
// follow each other with probability 'honestFollowProbability'.
// maliciousNbrFollow[i] gives the number of user that the ith
// malicisousUser should follow. maliciousNbrSybils[i] is the number
// of sybil nodes that the ith malicious user possesses.
func SetUpEntireCluster(t require.TestingT, trans transport.Transport,
	addr string, nbrHonest int, nbrMalicious int, honestFollowProability float64,
	maliciousNbrFollow []int, maliciousNbrSybils []int, opts ...Option) HonestNodeCluster {

	cluster := NewHonestNodeCluster(t, trans, addr, nbrHonest, opts...)
	cluster.SetUpFollowings(honestFollowProability)

	if nbrMalicious > 0 {
		require.Equal(t, len(maliciousNbrFollow), len(maliciousNbrSybils))
		// Add malicious
		for i := 0; i < nbrMalicious; i++ {
			cluster.AddMaliciousNode(t, trans, addr, maliciousNbrSybils[i], maliciousNbrFollow[i])
		}
	}
	return cluster
}

// Creates a cluster of honest nodes with 'n' nodes. Each node will
// send beetween 'minfFollowRequests' and 'maxFollowRequests' follow
// requests to other random nodes. A node can request to follow
// someone
// Total nbr of routing edges is approximately 'routingEdges'.
func NewHonestNodeCluster(t require.TestingT, trans transport.Transport,
	addr string, n int, opts ...Option) HonestNodeCluster {
	require.GreaterOrEqual(t, n, 1)
	nodes := make([]*HonestTestNode, n)
	for i := 0; i < n; i++ {
		node := NewHonestTestNode(t, trans, addr, opts...)
		// Chain the nodes
		if i > 0 {
			node.AddPeer(nodes[i-1].GetAddress())
		}
		nodes[i] = &node
	}
	// Close the chain (so that the graph is connected)
	nodes[0].AddPeer(nodes[n-1].GetAddress())
	fullyConnect(nodes)
	return HonestNodeCluster{
		nodes:          nodes,
		t:              t,
		maliciousNodes: make([]*MaliciousTestNode, 0),
	}
}

func (c *HonestNodeCluster) SetUpFollowings(followProbability float64) {
	if followProbability <= 0 {
		return
	}
	n := len(c.nodes)
	for i := 0; i < n-1; i++ {
		n1 := c.nodes[i]
		for j := i + 1; j < n; j++ {
			flip := rand.Float64()
			if flip <= followProbability {
				n2 := c.nodes[j]
				n1.Follow(n2.GetName())
			}
		}
	}
}

func (c *HonestNodeCluster) StopAll() {
	wg := sync.WaitGroup{}

	for _, node := range c.nodes {
		wg.Add(1)
		go func(n *HonestTestNode) {
			defer wg.Done()
			n.socket.Close()
			n.Stop()
		}(node)
	}

	for _, node := range c.maliciousNodes {
		wg.Add(1)
		go func(n *MaliciousTestNode) {
			defer wg.Done()
			n.socket.Close()
			n.Stop()
		}(node)
	}
	wg.Wait()
}

// Connect a malicious node to the cluster. The malicious node is peer of every honest nodes.
// The sybil nodes are not peer of any honest nodes. Add routing entry to sybil nodes to reach any
// honest node within the network.
func (c *HonestNodeCluster) AddMaliciousNode(t require.TestingT, trans transport.Transport,
	addr string, nbrChildren int, nFollowing int) {
	m := NewMaliciousTestNode(t, trans, addr)
	m.CreateSybilNodes(nbrChildren, t, trans, addr)
	children := m.GetChildren()
	// Add routes to honest nodes
	n := len(c.nodes)
	for i := 0; i < n; i++ {
		c.nodes[i].AddPeer(m.GetAddress())
		m.AddPeer(c.nodes[i].GetAddress())
		for j := 0; j < len(children); j++ {
			children[j].SetRoutingEntry(c.nodes[i].GetAddress(), m.GetAddress())
		}
	}

	// Same for malicious nodes
	for _, maliciousNode := range c.maliciousNodes {
		maliciousNode.AddPeer(m.GetAddress())
		m.AddPeer(maliciousNode.GetAddress())
		for _, child := range maliciousNode.GetChildren() {
			child.SetRoutingEntry(m.GetAddress(), maliciousNode.GetAddress())
		}
		for _, child := range children {
			child.SetRoutingEntry(maliciousNode.GetAddress(), m.GetAddress())
		}
	}

	// Follows exactly 'nFollowing' other nodes
	perm := rand.Perm(n + len(c.maliciousNodes))
	for i := 0; i < nFollowing; i++ {
		idx := perm[i]
		var toFollow string
		if idx < n {
			toFollow = c.nodes[idx].GetName()
		} else {
			toFollow = c.maliciousNodes[idx-n].GetName()
		}
		err := m.Follow(toFollow)
		require.NoError(c.t, err)
	}

	// Add malicious node to cluster
	c.maliciousNodes = append(c.maliciousNodes, &m)
}

// Randomly connect nodes by creating 'nbrEdges'.
func fullyConnect(nodes []*HonestTestNode) {
	n := len(nodes)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			nodes[i].AddPeer(nodes[j].GetAdress())
			nodes[j].AddPeer(nodes[i].GetAddress())
		}
	}
}

func (c *HonestNodeCluster) GetNodes() []*HonestTestNode {
	return c.nodes
}

func (c *HonestNodeCluster) GetMaliciousNodes() []*MaliciousTestNode {
	return c.maliciousNodes
}

// Returns the nbr of active nodes (ie. non-sybil)
func (c *HonestNodeCluster) NbrActiveNodes() int {
	return len(c.nodes) + len(c.maliciousNodes)
}
