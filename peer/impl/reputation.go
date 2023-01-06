package impl

import (
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
	err := n.Consensus(likerID, value, types.Reputation)
	// consensus failed
	if err != nil {
		return err
	}

	// consensus succeeded : tell everyone to update its table with the new score
	// broadcast a like or dislike message
	switch value {
	case "+1":
		// create a like message and broadcast it
		likeMsg := createLikeMsg(msgSender, msgID, score)
		likeMsgMarsh, err := n.reg.MarshalMessage(likeMsg)
		if err != nil {
			return err
		}
		err = n.Broadcast(likeMsgMarsh)
		return err
	case "-1":
		// create a dislike message and broadcast it
		likeMsg := createDisLikeMsg(msgSender, msgID, score)
		likeMsgMarsh, err := n.reg.MarshalMessage(likeMsg)
		if err != nil {
			return err
		}
		err = n.Broadcast(likeMsgMarsh)
		return err

	}
	return nil
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
