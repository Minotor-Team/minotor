package peer

type Reputation interface {
	InitReputationCheck(likerID string, value string, msgSender string, msgID string, score string) error
	Like(likerID string, value string, msgSender string, msgID string, score string) error
	Dislike(likerID string, value string, msgSender string, msgID string, score string) error
}
