package impl

import "sync"

type messageReputation struct {
	sync.RWMutex
	messageScore map[uint]uint
}

// initialize the messages reputation
func newMsgReputation() *messageReputation {
	return &messageReputation{
		messageScore: make(map[uint]uint),
	}
}
