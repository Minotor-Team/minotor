package impl

import (
	"time"

	"go.dedis.ch/cs438/datastructures"
	"go.dedis.ch/cs438/datastructures/concurrent"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

func (n *UserNode) GetSybilNodes() datastructures.Set[string] {
	return n.verifier.rejectedSuspects.Values()
}

func (n *UserNode) startProtocol(setup func(), protocol func() error) error {
	firstCheck := n.conf.SybilLimitFirstCheck
	if firstCheck == 0 {
		return nil
	}
	timer := time.NewTimer(firstCheck)
	<-timer.C
	err := protocol()
	if err != nil {
		return xerrors.Errorf("startSuspect: %v", err)
	}

	intervalCheck := n.conf.SybilLimitInterval
	if intervalCheck == 0 {
		return nil
	}

	ticker := time.NewTicker(intervalCheck)
	for {
		select {
		case <-ticker.C:
			setup()
			err := protocol()
			if err != nil {
				return xerrors.Errorf("startSuspect: %v", err)
			}
		case <-n.stopChannel:
			return nil
		}

	}
}

func (n *UserNode) ResetSuspect() {
	n.routeManager.Reset()
}

func (n *UserNode) StartSuspect() error {
	return n.startProtocol(n.ResetSuspect, n.SuspectSecureRandomRouteProtocol)
}

func (n *UserNode) StartVerifier() error {
	setup := func() {
		n.verifier.Restart()
		close(n.pendingSuspectTails)
		n.pendingSuspectTails = make(chan pendingSuspect)
		n.processedSuspects = concurrent.NewSet[string]()
	}
	protocol := func() error {
		err := n.VerifierSecureRandomRouteProtocol()
		if err != nil {
			return err
		}

		err = n.VerificationProtocol()
		if err != nil {
			return err
		}

		return nil
	}

	return n.startProtocol(setup, protocol)
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
	count := 0
	for ok {
		go func(c chan error) {
			_, err := n.StartRandomRoute(n.conf.RouteLength, routeID, n.conf.RouteTimeout, isSuspect)
			if err != nil {
				c <- err
			}
		}(errChan)
		count++
		routeID, ok = n.suspectRouteID.Next()
	}

	i := 0
	for err := range errChan {
		if err != nil {
			return nil, xerrors.Errorf("failed to execute the r routes: %v", err)
		}
		i++
		if i == count {
			break
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
	// The verifier is ready to verify.
	return nil
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
		if n.processedSuspects.Size() == n.conf.TotalPeers {
			close(n.pendingSuspectTails)
		}
	}

	return nil
}

func (n *UserNode) HandleVerfierRegistrationRequest(msg types.VerifierRegistrationQuery, pkt transport.Packet) error {
	tail := msg.Tail
	answer := &types.VerifierRegistrationAnswer{
		IsRegistered: false,
	}
	value, ok := n.routeManager.KeyStore.Get(msg.Suspect)
	if ok && value == tail.String() {
		answer.IsRegistered = true
	}
	addr := n.conf.Socket.GetAddress()
	header := transport.NewHeader(addr, addr, pkt.Header.Source, pkt.Header.TTL)
	err := n.createPktAndSend(&header, answer, pkt.Header.RelayedBy)
	if err != nil {
		return xerrors.Errorf("HandlerVerifierRegistrationRequest: failed to send back the answer : %v", err)
	}

	return nil
}

func (n *UserNode) HandleVerfierRegistrationAnswer(msg types.VerifierRegistrationAnswer) error {
	ok := n.verifier.notificationService.Notify(msg.InstanceNumber, msg.IsRegistered)
	if !ok {
		return xerrors.Errorf("HanlderVerifierRegistrationAnswer: notification error: can't notify instance #%d of registration", msg.InstanceNumber)
	}
	return nil
}

func (n *UserNode) isRegistered(suspect string, tail types.Edge, instanceNbr uint) (bool, error) {
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

			ok, err := n.isRegistered(v.Suspect, tail, instanceNbr)
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
		n.verifier.Accept(suspect)
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
