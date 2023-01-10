package impl

import (
	"fmt"
	"strconv"
	"time"

	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// A node has a reputation struct that corresponds to the number number of likes and number of dislikes he has
// (it would be the total of its messages). Moreover he has the ability to like or dislike messages of other peers
// and change their corresponding reputation score.

func (n *node) InitReputationCheck(likerID string, value int, msgSender string, msgID string, score string) (map[string]int, error) {
	if value != 1 && value != -1 {
		return nil, xerrors.Errorf("Wrong value, should be either +1 or -1 : %v", value)
	}
	err := n.LikeConsensus(likerID, value, msgSender, msgID)
	if err == nil {
		n.reg.ProcessScoreMap(n.messagesScore.messageScore)
	}
	return n.messagesScore.messageScore, err
}

func (n *node) LikeConsensus(likerID string, value int, msgSender string, msgID string) error {
	var isaLike bool
	if value == 1 {
		isaLike = true
	} else {
		isaLike = false
	}
	tp := types.Reputation
	name := likerID + "," + msgID + "," + strconv.Itoa(value)
	store := n.conf.Storage.GetReputationStore()

	storedValue := store.Get(name)
	// if the liker already liked (dislike) the msg ID, malicious user
	if string(storedValue) == strconv.Itoa(value) {
		fmt.Println("I have already liked, but I try 😈")
	}
	handler := n.reputationHandler

	// if only one node set name
	if n.conf.TotalPeers <= 1 {
		store.Set(name, []byte(strconv.Itoa(value)))
		n.messagesScore.updateMsgScore(msgID, isaLike)
		return nil
	}

	// loop on the paxos instance until consensus is finished
	startingStep := handler.getStep()
	for {
		step := handler.getStep()
		// if step different from initial one, restart consensus if value stored is not ours
		if startingStep != step {
			storedValue := store.Get(name)
			if string(storedValue) == strconv.Itoa(value) {
				return nil
			}
			return n.LikeConsensus(likerID, value, msgSender, msgID)
		}

		proposedValue := &types.PaxosLike{
			Name:  name,
			Value: value,
		}

		// launch paxos phase 2
		ret, err := n.Phase2(*proposedValue, name, value, handler, tp, msgSender, store)
		if err != nil || ret {
			return err
		}
	}
}

func (n *node) Phase2(value types.PaxosLike, name string, likeValue int, handler *paxosLikeHandler, tp types.PaxosType, msgSender string, store storage.Store) (bool, error) {
	// broadcast propose message with given value
	paxosProposeLike := handler.createProposeLike(value, tp)
	valueChannel := handler.getValueChannel()

	transportProposeLike, err := n.reg.MarshalMessage(&paxosProposeLike)
	if err != nil {
		return true, err
	}

	err = n.Broadcast(transportProposeLike)
	if err != nil {
		return true, err
	}

	ticker := time.NewTicker(n.conf.PaxosProposerRetry)
	// wait for either the value to be received or the timeout to occur
	select {
	case finalValue := <-valueChannel:
		// if final value is ours, everything went well
		if finalValue.Name == name && finalValue.Value == likeValue {
			return true, nil
		}
	case <-ticker.C:
		// timeout occurred
		fmt.Println("timeout")
		return true, nil
	}

	return false, nil
}

func (pH *paxosLikeHandler) createProposeLike(value types.PaxosLike, tp types.PaxosType) types.PaxosProposeLike {
	pH.RLock()
	defer pH.RUnlock()

	paxosProposeLike := types.PaxosProposeLike{
		Type:  tp,
		Step:  pH.step,
		ID:    pH.paxosID,
		Value: value,
	}

	return paxosProposeLike
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
