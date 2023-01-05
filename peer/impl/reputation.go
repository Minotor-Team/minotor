package impl

import (
	"go.dedis.ch/cs438/types"
)

// A node has a reputation struct that corresponds to the number number of likes and number of dislikes he has
// (it would be the total of its messages). Moreover he has the ability to like or dislike messages of other peers
// and change their corresponding reputation score.

func (n *node) InitReputationCheck(userID string, reputScore int64) error {
	// n.Consensus(userID, reputScore, types.Reputation)
	return n.Consensus("", "", types.Reputation)
}
