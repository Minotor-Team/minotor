package impl

import "sync"

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
