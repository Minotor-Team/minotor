package analysis

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport/channel"
)

func Test_Scenario_No_Malicious_User(t *testing.T) {
	trans := channel.NewTransport()
	cluster := z.SetUpEntireCluster(t, trans, "127.0.0.2:0", 100, 1, 0.4, []int{5}, []int{2})
	defer cluster.StopAll()

	m := cluster.GetMaliciousNodes()[0]
	err := m.AppendPublications([]string{"a", "b"}, []bool{true, false})
	require.NoError(t, err)
	err = m.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 120)
	for _, node := range cluster.GetNodes() {
		log.Info().Msgf("Honest node %v", node.GetAddress())
		require.Equal(t, 1, node.GetLiked().Size())
	}
	for _, node := range cluster.GetMaliciousNodes() {
		for _, child := range node.GetChildren() {
			log.Info().Msgf("Sybil node %v", node.GetAddress())
			require.Equal(t, 2, child.GetLiked().Size())
		}
		log.Info().Msgf("Malicious node %v", node.GetAddress())
		require.Equal(t, 2, node.GetLiked().Size())
	}
	//fmt.Printf("#liked: %v", cluster.GetNodes()[0].GetLiked().Size())
}
