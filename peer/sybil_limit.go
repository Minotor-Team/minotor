package peer

import (
	"time"

	"go.dedis.ch/cs438/types"
)

// TODO:
// Remove embedding, and add functionalities directly to the peer.
// Even if embedding allows to define clear layers, it is too hard to adapt the tests.
// The only exposed methods is getSybilNodes(). The SybilProtocol runs under the hood at a regular
// time interval.

// A SybilLimit node is a suspect and a verifier.
// It conforms to the two-phase protocol described in the paper.
type SybilLimitNode interface {
	RouteProtocol
	VerificationProtocol
}

type RouteProtocol interface {
	// Protocol executed by a suspect to initiate an s-instance.
	StartRandomRoute(budget uint, id uint, timeout time.Duration) (string, error)
	// Protocol executed by a node upon receiving a message corresponding to an s-instance.
	PropagateRandomRoute(msg types.RouteMessage, sender string) error
}

type SybilLimitProtocol interface {
	// Protocol executed by a node to initiate a SybilLimitProtocol.
	// The node broadcast its intention to start a sybil limit protocol.
	// When receving the launch message, it starts a secure random route protocol.
	// It then waits for it to stabilize (i.e., every response are acked), and then
	// it begins the verification protocol.
	LaunchSybilLimitProtocol()
	// Protocol executed by a node to initiate a secure random route protocol.
	// It executes r s-instance of random route and r v-instances of random route.
	SecureRandomRouteProtocol() error

	// Protocol executed by a node to initiate a verification protocol.
	// Must be used by a verifier that has received enough answer.
	// It computes a list of nodes that are believed to be sybils.
	VerificationProtocol() error
}

type VerificationProtocol interface{}

// A thread-safe interface to get social relations information.
type SocialProfil interface {
	// Ordered list of social relations.
	GetRelations() []string
	// Get the n-th relation.
	GetRelation(n uint) string
	// Get the index of the relation in the ordered list of relations.
	GetRelationIndex(relation string) (uint, bool)
	// Get the number of relations.
	NumberOfRelation() uint
	// Return the shared key with the provided relation, if any.
	GetSharedKey(relation string) (string, bool)
}
