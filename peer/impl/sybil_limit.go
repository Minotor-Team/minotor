package impl

import (
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// implements peer.SybilLimitProtocol
func (n *UserNode) SuspectSecureRandomRouteProtocol() error {
	tails, err := n.executeRoutes(true)
	if err != nil {
		return xerrors.Errorf("SuspectSecureRandomRouteProtocol: %v", err)
	}
	completionMsg := types.SuspectRouteProtocolDone{
		Tails: tails,
	}

	msg, err := n.conf.MessageRegistry.MarshalMessage(completionMsg)
	if err != nil {
		return xerrors.Errorf("SuspectSecureRandomRouteProtocol: failed to marshall the message: %v", err)
	}

	err = n.Broadcast(msg)
	if err != nil {
		return xerrors.Errorf("SuspectSecureRandomRouteProtocol: failed to broadcast the message: %v", err)
	}
	return nil
}

// Executes r route instance in parallel (and concurrently).
// Returns once every route completed.
func (n *UserNode) executeRoutes(isSuspect bool) (Tails, error) {
	var rangedID rangedIndex
	if isSuspect {
		rangedID = n.suspectRouteID
	} else {
		rangedID = n.verifierRouteID
	}

	routeID, ok := rangedID.Next()
	errChan := make(chan error)
	for ok {
		go func(c chan error) {
			_, err := n.StartRandomRoute(n.conf.RouteLength, routeID, n.conf.RouteTimeout, isSuspect)
			if err != nil {
				c <- err
			}
		}(errChan)

		routeID, ok = n.suspectRouteID.Next()
	}

	for err := range errChan {
		if err != nil {
			return nil, xerrors.Errorf("failed to execute the r routes: %v", err)
		}
	}

	tails := n.routeManager.TailStore.Entries()
	return rangedID.ExtractAll(tails), nil
}

// implements peer.SybilLimitProtocol
func (n *UserNode) VerifierSecureRandomRouteProtocol() error {
	tails, err := n.executeRoutes(false)
	if err != nil {
		return xerrors.Errorf("VerifierSecureRandomRouteProtocol: %v", err)
	}

	n.verifier.AddTails(tails)
	panic("To be implemented")
}

type pendingSuspect struct {
	Tails   map[uint]types.Edge
	Suspect string
}

func (n *UserNode) HandleSuspectRouteDone(msg types.SuspectRouteProtocolDone, sender string) error {
	if !n.processedSuspects.Contains(sender) {
		tails := pendingSuspect{
			Tails:   msg.Tails,
			Suspect: sender,
		}
		n.pendingSuspectTails <- tails
		n.processedSuspects.Add(sender)
	}

	return nil
}

// implements peer.SybilLimitProtocol
func (n *UserNode) VerificationProtocol() error {
	// Test is it can verify.
	v, ok := <-n.pendingSuspectTails
	for ok {
		// 1. The verifier receives the tails from suspects.
		suspectTails := v.Tails
		intersection := n.verifier.Intersection(suspectTails)
		instanceNumbers := make([]uint, 0)
		for instanceNbr, tail := range intersection {
			isRegistered := func(suspect string, tail types.Edge) bool {
				panic("Asks the end of the tail if the suspect is registered under To->From")
			}

			if isRegistered(v.Suspect, tail) {
				instanceNumbers = append(instanceNumbers, instanceNbr)
			}
		}
		if len(intersection) == 0 {
			panic("Reject the suspect")
		}

		instance, load, ok := n.verifier.MinimunLoadInstanceNumber(instanceNumbers)
		if !ok {
			panic("Reject the suspect")
		}

		if !n.verifier.SatisfiesSybilBound(load) {
			panic("Reject the suspect")
		}

		// The suspect satisfies the two conditions, we accept it.
		n.verifier.IncrementLoad(instance)
		panic("Accept the suspect")
		v, ok = <-n.pendingSuspectTails
	}
	panic("To be implemented")
}

type rangedIndex struct {
	low     uint
	current uint
	high    uint
}

func NewRangedIndex(low uint, high uint) *rangedIndex {
	return &rangedIndex{low: low, current: low, high: high}
}

func (r *rangedIndex) Next() (uint, bool) {
	if r.current == r.high {
		return 0, false
	} else {
		res := r.current
		r.current++
		return res, true
	}
}

func (r *rangedIndex) Size() uint {
	return r.high - r.low
}

func (r *rangedIndex) Contains(value uint) bool {
	return r.low <= value && value < r.high
}

func (r *rangedIndex) ExtractAll(from Tails) Tails {
	res := make(Tails)
	for k, v := range from {
		if r.Contains(k) {
			res[k] = v
		}
	}

	return res
}
