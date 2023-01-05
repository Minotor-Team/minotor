package peer

type Reputation interface {
	InitReputationCheck(userID string, reputScore int) error
}
