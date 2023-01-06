package peer

type Reputation interface {
	InitReputationCheck(likerID string, value string, msgSender string, msgID string, score string) error
}
