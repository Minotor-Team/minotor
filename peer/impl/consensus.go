package impl

import (
	"fmt"
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// maps key to given value
func (n *node) Consensus(key string, value string, tp types.PaxosType) error {
	var store storage.Store
	var handler *paxosHandler

	switch tp {
	case types.Tag:
		store = n.conf.Storage.GetNamingStore()
		handler = n.tagHandler
	case types.Identity:
		store = n.conf.Storage.GetIdentityStore()
		handler = n.identityHandler
	default:
		return xerrors.Errorf("invalid type : %v", tp)
	}

	// check if key not already in the store
	storedMh := store.Get(key)
	if storedMh != nil {
		return xerrors.Errorf("already existing name : %v", key)
	}

	// if only one node set key/value
	if n.conf.TotalPeers <= 1 && tp == types.Tag {
		store.Set(key, []byte(value))
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
		// if step different from initial one, restart consensus if value stored is not ours
		if startingStep != step {
			storedMh = store.Get(key)
			if storedMh != nil {
				return nil
			}
			return n.Consensus(key, value, tp)
		}

		// launch paxos phase 1
		proposedValue, err := n.PaxosPhase1(key, value, handler)
		if (err != nil || *proposedValue == types.PaxosValue{}) {
			return err
		}

		// launch paxos phase 2
		ret, err := n.PaxosPhase2(*proposedValue, key, value, handler, store)
		if err != nil || ret {
			return err
		}
	}
}

// starts paxos phase 1
func (n *node) PaxosPhase1(key string, value string, handler *paxosHandler) (*types.PaxosValue, error) {
	for {
		myAddr := n.soc.GetAddress()

		// broadcast pepare message with proposed value
		proposedValue := types.PaxosValue{
			UniqID:   xid.New().String(),
			Filename: key,
			Metahash: value,
		}

		paxosPrepareMsg := handler.createPrepareMessage(myAddr, proposedValue)
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
func (n *node) PaxosPhase2(paxosValue types.PaxosValue, key string, value string, handler *paxosHandler, store storage.Store) (bool, error) {

	// broadcast propose message with given value
	paxosProposeMsg := handler.createProposeMessage(paxosValue)
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
		if finalValue.Filename == key && finalValue.Metahash == value {
			return true, nil
		}
		// else start again with consensus function
		fmt.Printf("a) value : %v\n", finalValue)
		return true, n.Consensus(key, value, handler.getType())
	// if ticker is triggered
	case <-ticker.C:
		// check anyway if final value is not nil (refer previous case)
		finalValue := handler.getFinalValue()
		if (finalValue != types.PaxosValue{}) {
			if finalValue.Filename == key && finalValue.Metahash == value {
				return true, nil
			}
			fmt.Printf("b) value : %v\n", finalValue)
			return true, n.Consensus(key, value, handler.getType())
		}
		// increment paxos ID and restart loop
		handler.nextID()
	// if stop function is called, stop function
	case <-n.stopChannel:
		return true, nil
	}
	return false, nil
}
