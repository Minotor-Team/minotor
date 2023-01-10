package peer

import (
	"sync"
	"time"

	"go.dedis.ch/cs438/datastructures"
	"go.dedis.ch/cs438/types"
)

type SybilVerifier interface {
	// Returns a list of potential sybil nodes
	GetSybilNodes() datastructures.Set[string]
}

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

	// Protocol executed by a suspect to initiate a secure random route protocol.
	// It executes r s-instance of random route.
	// By convention, the suspect identifies the route with ids in [r, 2r - 1].
	SuspectSecureRandomRouteProtocol() error

	// Protocol executed by a verifier to initiate a secure random route protocol.
	// It executes r v-instance of random route.
	// By convention, the verifier identifies the route with ids in [0, r-1].
	VerifierSecureRandomRouteProtocol() error

	// Describes the behavior when the verifier receives a SuspectRouteProtocolDone message.
	HandleSuspectRouteProtocol(msg types.SuspectRouteProtocolDone, sender string) error

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

type SocialProfilImpl struct {
	lock         sync.RWMutex
	relations    []string
	indexMapping map[string]int
}

func NewSocialProfilAdapter() *SocialProfilImpl {
	return &SocialProfilImpl{
		relations:    []string{},
		indexMapping: make(map[string]int),
	}
}

func (p *SocialProfilImpl) GetRelations() []string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.relations
}

func (p *SocialProfilImpl) GetRelation(n uint) string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.relations[n]
}

func (p *SocialProfilImpl) GetRelationIndex(relation string) (uint, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	index, ok := p.indexMapping[relation]
	return uint(index), ok
}

func (p *SocialProfilImpl) NumberOfRelation() uint {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return uint(len(p.relations))
}

func (p *SocialProfilImpl) GetSharedKey(relation string) (string, bool) {
	return "SHARED_KEY", true
}
