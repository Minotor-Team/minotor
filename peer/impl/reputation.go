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
	return nil
}

func (n *node) Like(likerID string, value string, msgSender string, msgID string, score string) error {
	msg := createLikeMsg(msgSender, msgID, score)
	err := n.BroadcastLike(msg)
	if err != nil {
		return err
	}
	handler := n.reputationHandler
}

func (n *node) Dislike(likerID string, value string, msgSender string, msgID string, score string) error {
	msg := createDisLikeMsg(msgSender, msgID, score)
	err := n.BroadcastDisLike(msg)
}

func (n *node) BroadcastLike(msg *types.LikeMessage) error {
	msgMarsh, err := n.reg.MarshalMessage(msg)
	if err != nil {
		return err
	}

	// broadcast accept message
	err = n.Broadcast(msgMarsh)
	return err
}

func (n *node) BroadcastDisLike(msg *types.DislikeMessage) error {
	msgMarsh, err := n.reg.MarshalMessage(msg)
	if err != nil {
		return err
	}

	// broadcast accept message
	err = n.Broadcast(msgMarsh)
	return err
}
func createLikeMsg(msgSender string, msgID string, score string) *types.LikeMessage {
	return &types.LikeMessage{
		MsgSenderID: msgSender,
		MsgID:       msgID,
		Score:       score,
	}
}
func createDisLikeMsg(msgSender string, msgID string, score string) *types.DislikeMessage {
	return &types.DislikeMessage{
		MsgSenderID: msgSender,
		MsgID:       msgID,
		Score:       score,
	}
}
