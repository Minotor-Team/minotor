package types

import (
	"strings"

	"golang.org/x/xerrors"
)

type RouteMessage struct {
	// Keeps track of the length of the route
	Length uint
	// Budget of the route, i.e. the number of hops it can take before it is completed.
	Budget uint
	// Data transmitted by the node
	Data string
	// A MAC to authenticate the message.
	MAC string
	// An unique identifier that corresponds to a given s-instance (or v-instance).
	// For r differents v-instances, use a numbers in range [1, r]
	ID uint
	// A boolean flag to indicate the direction of the message.
	// A reversed message should use the reverse routing permutation.
	Reversed bool

	// A flag to indicate if the last hop should register the data or not.
	MustRegister bool
}

type SuspectRouteProtocolDone struct {
	Tails map[uint]Edge
}

// A message to query a node if it has registered the suspect under
// the given tail.
type VerifierRegistrationQuery struct {
	Suspect string
	Tail    Edge
}

type VerifierRegistrationAnswer struct {
	Suspect      string
	Tail         Edge
	IsRegistered bool
}

const EdgeSep string = "->"

type Edge struct {
	From string
	To   string
}

// Two edges intersects if they correspond to the same undirected edge in the graph
func (e *Edge) Intersects(that Edge) bool {
	return (e.From == that.From && e.To == that.To) || (e.From == that.To && e.To == that.From)
}

func (e *Edge) String() string {
	return strings.Join([]string{e.From, e.To}, EdgeSep)
}

func (e *Edge) Reverse() Edge {
	return Edge{From: e.To, To: e.From}
}

func ParseEdge(str string) (Edge, error) {
	nodes := strings.Split(str, EdgeSep)
	if len(nodes) != 2 {
		return Edge{}, xerrors.Errorf("ToEdge: %s is not an edge", str)
	}

	return Edge{From: nodes[0], To: nodes[1]}, nil
}
