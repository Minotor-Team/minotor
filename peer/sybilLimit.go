package peer

type SybilVerifier interface {
	// Returns a list of potential sybil nodes
	GetSybilNodes() []string
}
