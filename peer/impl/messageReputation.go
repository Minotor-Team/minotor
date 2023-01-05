package impl

import "sync"

type messageReputation struct {
	sync.RWMutex
	messageScore map[int64]int64
}

// initialize the messages reputation
func newMsgReputation() *messageReputation {
	return &messageReputation{
		messageScore: make(map[int64]int64),
	}
}

// given
func (reputation *messageReputation) updateMessageReputation(msgID int64, like bool) {
	reputation.Lock()
	defer reputation.Unlock()
	if like {
		reputation.messageScore[msgID]++
	} else {
		reputation.messageScore[msgID]--
	}
}
