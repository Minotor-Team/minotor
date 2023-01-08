package impl

import (
	"fmt"

	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// A node has a reputation struct that corresponds to the number number of likes and number of dislikes he has
// (it would be the total of its messages). Moreover he has the ability to like or dislike messages of other peers
// and change their corresponding reputation score.

func (n *node) InitReputationCheck(likerID string, value string, msgSender string, msgID string, score string) error {
	if value != "+1" && value != "-1" {
		return xerrors.Errorf("Wrong value, should be either +1 or -1 : %v", value)
	}
	fmt.Println("Before : ", n.conf.Storage.GetReputationStore().Get(likerID+","+msgID))
	fmt.Println("Before : ", n.conf.Storage.GetReputationStore().Len())
	err := n.Consensus(likerID+","+msgID, value, types.Reputation)
	fmt.Println("Before : ", n.conf.Storage.GetReputationStore().Len())
	fmt.Println("After : ", n.conf.Storage.GetReputationStore().Get(likerID+","+msgID))

	return err
}

func createLikeMsg(msgSender string, msgID string, score string) types.LikeMessage {
	return types.LikeMessage{
		MsgSenderID: msgSender,
		MsgID:       msgID,
		Score:       score,
	}
}
func createDisLikeMsg(msgSender string, msgID string, score string) types.DislikeMessage {
	return types.DislikeMessage{
		MsgSenderID: msgSender,
		MsgID:       msgID,
		Score:       score,
	}
}
