package performance

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/performance"
	"go.dedis.ch/cs438/transport/channel"
)

func Test_Performance(t *testing.T) {
	transp := channel.NewTransport()
	n1 := z.NewHonestTestNode(t, transp, "127.0.0.1:0")
	defer n1.Stop()

	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	n1.AddPeer(n2.GetAddress())
	//n2.AddPeer(n1.GetAdress())

	p0 := peer.Publication{
		ID:      "0",
		Author:  n2.GetName(),
		Content: "HELLO",
	}
	p := performance.BinaryPublication{
		Publication: p0,
		TruthValue:  true,
	}
	err := n2.Publish(p)
	log.Info().Msgf("all outs: %v", n2.GetOuts())
	require.NoError(t, err)
}

func Test_Malicious_Node_Likes_Own_Publication(t *testing.T) {
	transp := channel.NewTransport()

	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	contents := []string{"a", "b"}
	tValues := []bool{true, false}
	err := n2.AppendPublications(contents, tValues)
	require.NoError(t, err)

	err = n2.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	published := n2.GetPublished()
	reactions := n2.GetReactions()[n2.GetName()]
	require.Equal(t, len(published), len(reactions))

	for _, p := range published {
		require.Contains(t, reactions, p.ID)
		// Every publications should be liked
		require.Equal(t, reactions[p.ID], performance.LIKE)
	}
}

func Test_Malicious_Node_Rejects_Others_Publications(t *testing.T) {
	transp := channel.NewTransport()

	n1 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n1.Stop()

	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	n1.AddPeer(n2.GetAddress())
	n2.AddPeer(n1.GetAddress())

	contents := []string{"a", "b"}
	tValues := []bool{true, false}

	err := n1.AppendPublications(contents, tValues)
	require.NoError(t, err)

	err = n1.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	published := n1.GetPublished()
	reactions := n2.GetReactions()[n1.GetName()]
	require.Equal(t, len(published), len(reactions))

	for _, p := range published {
		require.Contains(t, reactions, p.ID)
		// Every publications should be liked
		require.Equal(t, reactions[p.ID], performance.DISLIKE)
	}
}

func Test_Honest_Node_Are_Honest(t *testing.T) {
	transp := channel.NewTransport()

	n1 := z.NewHonestTestNode(t, transp, "127.0.0.1:0")
	defer n1.Stop()

	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	n1.AddPeer(n2.GetAddress())
	n2.AddPeer(n1.GetAddress())

	contents := []string{"a", "b"}
	tValues := []bool{true, false}
	err := n2.AppendPublications(contents, tValues)
	require.NoError(t, err)

	err = n2.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	published := n2.GetPublished()
	reactions := n1.GetReactions()[n2.GetName()]
	require.Equal(t, len(published), len(reactions))

	for _, p := range published {
		require.Contains(t, reactions, p.ID)
		// Every publications should be liked
		require.Equal(t, reactions[p.ID], p.TruthValue)
	}
}

func Test_Sybil_Nodes_Likes_Parent_Publications_And_Dislikes_Others(t *testing.T) {
	transp := channel.NewTransport()

	n1 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n1.Stop()

	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	n3 := z.NewSybilTestNode(t, transp, "127.0.0.1:0", &n1)
	defer n3.Stop()

	n1.AddPeer(n2.GetAddress())
	n2.AddPeer(n1.GetAddress())

	contents := []string{"a", "b"}
	tValues := []bool{true, false}

	err := n1.AppendPublications(contents, tValues)
	require.NoError(t, err)

	err = n2.AppendPublications(contents, tValues)
	require.NoError(t, err)

	err = n1.Attack()
	require.NoError(t, err)

	err = n2.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	p1 := n1.GetPublished()
	p2 := n2.GetPublished()
	r := n3.GetReactions()
	r1 := r[n1.GetName()]
	r2 := r[n2.GetName()]

	require.Equal(t, len(p1), len(r1))
	require.Equal(t, len(p2), len(r2))

	for _, p := range p1 {
		require.Contains(t, r1, p.ID)
		// Every publications from parent should be liked
		require.Equal(t, r1[p.ID], performance.LIKE)
	}

	for _, p := range p2 {
		require.Contains(t, r2, p.ID)
		// Every publications from other than parent should be disliked
		require.Equal(t, r2[p.ID], performance.DISLIKE)
	}
}

func Test_Sybil_Nodes_Automatically_Follows_Parent_When_CreateSybilNodes_Is_Used(t *testing.T) {
	transp := channel.NewTransport()
	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	defer n2.Stop()

	n2.CreateSybilNodes(1, t, transp, "127.0.0.1:0")
	children := n2.GetChildren()
	n3 := children[0]

	time.Sleep(time.Second)
	f2 := n2.GetFollowed()
	f3 := n3.GetFollowed()
	require.Equal(t, 1, f2.Size())
	require.Equal(t, 1, f3.Size())
	require.Equal(t, true, f2.Contains(n3.GetName()))
	require.Equal(t, true, f3.Contains(n2.GetName()))
}

func Test_Cluster_Is_Connected(t *testing.T) {
	transp := channel.NewTransport()
	n := 10
	cluster := z.NewHonestNodeCluster(t, transp, "127.0.0.1:0", n)
	defer cluster.StopAll()

	time.Sleep(time.Second * 10)

	nodes := cluster.GetNodes()
	for i := 0; i < n; i++ {
		require.Equal(t, n, len(nodes[i].GetRoutingTable()))
	}

	// cluster.SetUpFollowings(0, 3)
	// time.Sleep(time.Second * 5)
	// for i := 0; i < n; i++ {
	// 	log.Info().Msgf("%v: followed = %v", i, nodes[i].GetFollowed())
	// }
}

func Test_Cluster_Followings_Are_Reciprocal(t *testing.T) {
	transp := channel.NewTransport()
	n := 10
	cluster := z.NewHonestNodeCluster(t, transp, "127.0.0.1:0", n)
	defer cluster.StopAll()

	nodes := cluster.GetNodes()

	m := make(map[string]int)
	for idx, node := range nodes {
		m[node.GetName()] = idx
	}

	cluster.SetUpFollowings(0.2)
	time.Sleep(time.Second * 5)
	for i := 0; i < n; i++ {
		node := nodes[i]
		followed := node.GetFollowed()
		for _, user := range followed.ToArray() {
			userNode := nodes[m[user]]
			userFollowed := userNode.GetFollowed()
			// Check that user follow nodes[i]
			require.Equal(t, true, userFollowed.Contains(node.GetName()))
		}
	}
}

func Test_Add_Malicious_Node_Works_Correctly(t *testing.T) {
	transp := channel.NewTransport()
	n := 10
	cluster := z.NewHonestNodeCluster(t, transp, "127.0.0.1:0", n)
	defer cluster.StopAll()

	cluster.SetUpFollowings(0.3)
	time.Sleep(time.Second)

	// malicious := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0")
	// sybils := malicious.CreateSybilNodes(5, t, transp, "127.0.0.1:0")

	nbrFollowed := 4
	nbrChildren := 5
	cluster.AddMaliciousNode(t, transp, "127.0.0.1:0", nbrChildren, nbrFollowed)
	time.Sleep(time.Second * 2)

	// Exactly 1 malicious node in cluster
	require.Equal(t, 1, len(cluster.GetMaliciousNodes()))
	malicious := cluster.GetMaliciousNodes()[0]

	// Malicous node should follows exactly 'nbrFollowed + len(sybils)'
	require.Equal(t, malicious.GetFollowed().Size(), nbrFollowed+nbrChildren)

	// Malicious node should follow exactly 'nbrFollowed' non-children nodes
	followed := malicious.GetFollowed()
	for _, s := range malicious.GetChildren() {
		followed.Remove(s.GetName())
	}
	require.Equal(t, nbrFollowed, followed.Size())

	// Sybils have a route to everyone

	// Add another malicious and test again
	nbrFollowed = 7
	nbrChildren = 12
	cluster.AddMaliciousNode(t, transp, "127.0.0.1:0", nbrChildren, nbrFollowed)
	time.Sleep(time.Second * 2)

	// Exactly 2 malicious node in cluster
	require.Equal(t, 2, len(cluster.GetMaliciousNodes()))
	malicious = cluster.GetMaliciousNodes()[1]

	// Malicous node should follows exactly 'nbrFollowed + len(sybils)'
	require.Equal(t, malicious.GetFollowed().Size(), nbrFollowed+nbrChildren)

	// Malicious node should follow exactly 'nbrFollowed' non-children nodes
	followed = malicious.GetFollowed()
	for _, s := range malicious.GetChildren() {
		followed.Remove(s.GetName())
	}
	require.Equal(t, nbrFollowed, followed.Size())

	// Sybils should have route to each non-sybil node
	nActiveNodes := cluster.NbrActiveNodes()
	for _, m := range cluster.GetMaliciousNodes() {
		for _, s := range m.GetChildren() {
			require.Equal(t, nActiveNodes+1, len(s.GetRoutingTable())) // +1 to account for itself
			// Only 1 follow
			require.Equal(t, 1, s.GetFollowed().Size())
		}
	}
}

// func Test_Connecting_Routes(t *testing.T) {
// 	transp := channel.NewTransport()
// 	n1 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0", z.WithHeartbeat(time.Millisecond*50))
// 	//defer n1.Stop()

// 	n2 := z.NewMaliciousTestNode(t, transp, "127.0.0.1:0", z.WithHeartbeat(time.Millisecond*50))
// 	//defer n2.Stop()

// 	n3 := z.NewSybilTestNode(t, transp, "127.0.0.1:0", &n1, z.WithHeartbeat(time.Millisecond*50))
// 	//defer n3.Stop()

// 	n1.AddPeer(n2.GetAddress())
// 	n2.AddPeer(n3.GetAddress())
// 	n3.AddPeer(n1.GetAddress())

// 	time.Sleep(time.Second * 2)
// 	fmt.Printf("%v\n\n", n1.GetRoutingTable())
// 	fmt.Printf("%v\n\n", n2.GetRoutingTable())
// 	fmt.Printf("%v\n\n", n3.GetRoutingTable())

// }
