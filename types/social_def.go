package types

type PublicationMessage struct {
	ID      string
	Author  string
	Content string
}

type FollowRequest struct {
	RequestID string
	Source    string //username
}

type FollowResponse struct {
	RequestID string
}

// Define all other messages used for the reputation system
// Has to define a PaxosValue + every Paxos Message type
