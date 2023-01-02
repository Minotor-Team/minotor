package types

// reputation !
type ReputationValue struct {
	// ID of the node
	nodeID string

	// map between the mess IDs and the nodes ID and their score
	// Ex : the message ID 100 has 2 likes and 1 Dislike and the network has 5 peers
	// message ID 3 has 1 like only.
	// We have {mess100 : {node0 : 1, node1 : 0, node2 : -1, node3 : 1, node4 : 0},
	// mess3 : {node0 : 0, node1 : 0, node2 : 0, node3 : 1, node4 : 0}}
	messVotes map[string]map[string]uint
}

// reputation !
type ReputationBlock struct {
	// Index in the block of the reputation blockchain
	Index uint

	// SHA256(Index || value.nodeID || value.messVotes.keys || PrevHash)
	Hash []byte

	Value ReputationValue

	// Hash of the previous block
	PrevHash []byte
}
