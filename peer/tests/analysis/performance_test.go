package analysis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport/channel"
)

func Test_Scenario_No_Malicious_User(t *testing.T) {
	trans := channel.NewTransport()
	cluster := z.SetUpEntireCluster(t, trans, "127.0.0.1:0", 100, 1, 0.4, []int{10}, []int{15})
	defer cluster.StopAll()

	m := cluster.GetMaliciousNodes()[0]
	err := m.AppendPublications([]string{"a", "b"}, []bool{true, false})
	require.NoError(t, err)
	err = m.Attack()
	require.NoError(t, err)

	time.Sleep(time.Second * 10)

}
