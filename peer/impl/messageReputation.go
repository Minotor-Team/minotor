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

type messagesScore struct {
	sync.RWMutex
	messageScore map[string]int
}

// initialize the messages score map
func newMsgScore() *messagesScore {
	return &messagesScore{
		messageScore: make(map[string]int),
	}
}

func (reputation *messagesScore) updateMsgScore(msgID string, like bool) {
	reputation.Lock()
	defer reputation.Unlock()
	if like {
		reputation.messageScore[msgID]++
	} else {
		reputation.messageScore[msgID]--
	}
}
