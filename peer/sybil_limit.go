package peer

import "go.dedis.ch/cs438/types"

// A SybilLimit node is a suspect and a verifier.
// It conforms to the two-phase protocol described in the paper.
type SybilLimitNode interface {
	RouteProtocol
	VerificationProtocol
}

type RouteProtocol interface {
	// Protocol executed by a suspect to initiate an s-instance.
	StartRandomRoute(budget uint, id uint) error
	// Protocol executed by a node upon receiving a message corresponding to an s-instance.
	PropagateRandomRoute(msg types.Message) error
}

type VerificationProtocol interface{}
