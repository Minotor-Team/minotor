package impl

import (
	"go.dedis.ch/cs438/datastructures"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

func (n *UserNode) GetSybilNodes() datastructures.Set[string] {
	return n.verifier.rejectedSuspects
}

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
	v, isChannelOpen := <-n.pendingSuspectTails
	for isChannelOpen {
		// 1. The verifier receives the tails from suspects.
		suspect := v.Suspect
		suspectTails := v.Tails
		intersection := n.verifier.Intersection(suspectTails)
		instanceNumbers := make([]uint, 0)
		for instanceNbr, tail := range intersection {
			isRegistered := func(suspect string, tail types.Edge) (bool, error) {
				query := types.VerifierRegistrationQuery{
					Suspect: suspect,
					Tail:    tail,
				}

				unicast := func() error {
					dest := tail.To
					msg, err := n.conf.MessageRegistry.MarshalMessage(query)
					if err != nil {
						return xerrors.Errorf("failed to marshall the message: %v", err)
					}

					return n.Unicast(dest, msg)
				}

				isRegistered, err := n.verifier.notificationService.ExecuteAndWait(unicast, instanceNbr, n.conf.VerifierRegistrationTimeout)
				if err != nil {
					return isRegistered, xerrors.Errorf("isRegistered: %v", err)
				}
				return isRegistered, nil
			}
			ok, err := isRegistered(v.Suspect, tail)
			if err != nil {
				return xerrors.Errorf("VerificationProtocol: %v", err)
			}
			if ok {
				instanceNumbers = append(instanceNumbers, instanceNbr)
			}
		}
		if len(intersection) == 0 {
			n.verifier.Reject(suspect)
		}

		instance, load, ok := n.verifier.MinimunLoadInstanceNumber(instanceNumbers)
		if !ok {
			n.verifier.Reject(suspect)
		}

		if !n.verifier.SatisfiesSybilBound(load) {
			n.verifier.Reject(suspect)
		}

		// The suspect satisfies the two conditions, we accept it.
		n.verifier.IncrementLoad(instance)
		v, isChannelOpen = <-n.pendingSuspectTails
	}

	return nil
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
