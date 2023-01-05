package impl

import (
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// maps name to given metahash
func (n *node) Consensus(name string, mh string, tp types.PaxosType) error {
	var store storage.Store
	var handler *paxosHandler
	switch tp {
	case types.Tag:
		store = n.conf.Storage.GetNamingStore()
		// check if name not already in the naming store
		storedName := store.Get(name)
		if storedName != nil {
			return xerrors.Errorf("already existing name : %v", name)
		}
		handler = n.tagHandler
	case types.Identity:
		store = n.conf.Storage.GetIdentityStore()
		handler = n.identityHandler
	case types.Reputation:
		store = n.conf.Storage.GetReputationStore()
		handler = n.reputationHandler
	default:
		return xerrors.Errorf("invalid type : %v", tp)
	}

	// if only one node set name
	if n.conf.TotalPeers <= 1 {
		store.Set(name, []byte(mh))
		return nil
	}

	// wait for paxos instance to finish
	instanceRunning, instanceChannel := handler.initPaxosInstance()
	for instanceRunning {
		select {
		case <-instanceChannel:
			instanceRunning, instanceChannel = handler.initPaxosInstance()
		case <-n.stopChannel:
			return nil
		}
	}

	// loop on the paxos instance until consensus is finished
	startingStep := handler.getStep()
	for {
		step := handler.getStep()
		// if step different from initial one, restart tag if value stored is not ours
		if startingStep != step {
			storedName := store.Get(name)
			if storedName != nil {
				return nil
			}
			return n.Tag(name, mh)
		}

		// launch paxos phase 1
		proposedValue, err := n.PaxosPhase1(name, mh, handler, tp)
		if (err != nil || *proposedValue == types.PaxosValue{}) {
			return err
		}

		// launch paxos phase 2
		ret, err := n.PaxosPhase2(*proposedValue, name, mh, handler, tp, store)
		if err != nil || ret {
			return err
		}
	}
}

// starts paxos phase 1
func (n *node) PaxosPhase1(name string, mh string, handler *paxosHandler, tp types.PaxosType) (*types.PaxosValue, error) {
	for {
		myAddr := n.soc.GetAddress()

		// broadcast pepare message with proposed value
		proposedValue := types.PaxosValue{
			UniqID:   xid.New().String(),
			Filename: name,
			Metahash: mh,
		}

		paxosPrepareMsg := handler.createPrepareMessage(myAddr, proposedValue, tp)
		promiseChannel := handler.getPromiseChannel()

		transportPrepareMsg, err := n.reg.MarshalMessage(&paxosPrepareMsg)
		if err != nil {
			return nil, err
		}

		err = n.Broadcast(transportPrepareMsg)
		if err != nil {
			return nil, err
		}

		// create ticker with given interval
		ticker := time.NewTicker(n.conf.PaxosProposerRetry)
		select {
		// if promise channel is trigger, return value
		case value := <-promiseChannel:
			return &value, nil
		// if ticker is triggered, increment paxos ID
		case <-ticker.C:
			handler.nextID()
		// if stop function is called, stop function
		case <-n.stopChannel:
			return &types.PaxosValue{}, nil
		}
	}
}

// starts paxos phase 2
func (n *node) PaxosPhase2(value types.PaxosValue, name string, mh string, handler *paxosHandler, tp types.PaxosType, store storage.Store) (bool, error) {
	// broadcast propose message with given value
	paxosProposeMsg := handler.createProposeMessage(value, tp)
	valueChannel := handler.getValueChannel()

	transportProposeMsg, err := n.reg.MarshalMessage(&paxosProposeMsg)
	if err != nil {
		return true, err
	}

	err = n.Broadcast(transportProposeMsg)
	if err != nil {
		return true, err
	}

	// create ticker with given interval
	ticker := time.NewTicker(n.conf.PaxosProposerRetry)
	select {
	// if value channel is trigger
	case finalValue := <-valueChannel:
		// if final value is ours, everything went well
		if finalValue.Filename == name && finalValue.Metahash == mh {
			return true, nil
		}
		// else start again with tag function
		return true, n.Tag(name, mh)
	// if ticker is triggered
	case <-ticker.C:
		// check anyway if final value is not nil (refer previous case)
		finalValue := handler.getFinalValue()
		if (finalValue != types.PaxosValue{}) {
			if finalValue.Filename == name && finalValue.Metahash == mh {
				return true, nil
			}
			return true, n.Tag(name, mh)
		}
		// increment paxos ID and restart loop
		handler.nextID()
	// if stop function is called, stop function
	case <-n.stopChannel:
		return true, nil
	}
	return false, nil
}
