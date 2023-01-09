package peer

type Reputation interface {
	InitReputationCheck(likerID string, value int, msgSender string, msgID string, score string) (map[string]int, error)
}
