package impl

import (
	"crypto"
	"encoding/hex"
	"strconv"
	"sync"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
)

const paxosPhase1 = 1
const paxosPhase2 = 2
const initValue = 0
const running = true
const idle = false

// initialises paxosHandler
func newPaxosHandler(conf peer.Configuration) *paxosHandler {
	return &paxosHandler{
		running:         idle,
		step:            initValue,
		maxID:           initValue,
		bestID:          initValue,
		acceptedID:      initValue,
		promises:        initValue,
		currentPhase:    paxosPhase1,
		paxosID:         conf.PaxosID,
		nPeers:          conf.TotalPeers,
		threshold:       uint(conf.PaxosThreshold(conf.TotalPeers)),
		bestValue:       types.PaxosValue{},
		acceptedValue:   types.PaxosValue{},
		proposedValue:   types.PaxosValue{},
		finalValue:      types.PaxosValue{},
		acceptedValues:  make(map[types.PaxosValue]uint),
		TLCMap:          make(map[uint][]types.TLCMessage),
		promiseChannel:  make(chan types.PaxosValue, 1),
		valueChannel:    make(chan types.PaxosValue, 1),
		instanceChannel: make(chan struct{}),
	}
}

type paxosHandler struct {
	sync.RWMutex
	running         bool
	step            uint
	maxID           uint
	bestID          uint
	acceptedID      uint
	promises        uint
	currentPhase    uint
	paxosID         uint
	nPeers          uint
	threshold       uint
	bestValue       types.PaxosValue
	acceptedValue   types.PaxosValue
	proposedValue   types.PaxosValue
	finalValue      types.PaxosValue
	acceptedValues  map[types.PaxosValue]uint
	TLCMap          map[uint][]types.TLCMessage
	promiseChannel  chan types.PaxosValue
	valueChannel    chan types.PaxosValue
	instanceChannel chan struct{}
}

// inits paxos instance
func (pH *paxosHandler) initPaxosInstance() (bool, chan struct{}) {
	pH.Lock()
	defer pH.Unlock()

	// if paxos instance already running, return instance channel
	// that will be triggered when instance will be finished
	if pH.running {
		return running, pH.instanceChannel
	}

	// otherwise start instance
	pH.running = running
	return idle, nil
}

// clears paxos instance by resetting paramaters
func (pH *paxosHandler) clearInstance(conf peer.Configuration) {
	pH.running = idle
	pH.maxID = initValue
	pH.bestID = initValue
	pH.acceptedID = initValue
	pH.promises = initValue
	pH.currentPhase = paxosPhase1
	pH.paxosID = conf.PaxosID
	pH.nPeers = conf.TotalPeers
	pH.threshold = uint(conf.PaxosThreshold(conf.TotalPeers))
	pH.bestValue = types.PaxosValue{}
	pH.acceptedValue = types.PaxosValue{}
	pH.proposedValue = types.PaxosValue{}
	pH.acceptedValues = make(map[types.PaxosValue]uint)
	pH.TLCMap = make(map[uint][]types.TLCMessage)
	pH.promiseChannel = make(chan types.PaxosValue, 1)
	pH.valueChannel <- pH.finalValue
	close(pH.valueChannel)
	pH.valueChannel = make(chan types.PaxosValue, 1)
	close(pH.instanceChannel)
	pH.instanceChannel = make(chan struct{})
}

// gets current step
func (pH *paxosHandler) getStep() uint {
	pH.RLock()
	defer pH.RUnlock()

	return pH.step
}

// gets final value
func (pH *paxosHandler) getFinalValue() types.PaxosValue {
	pH.RLock()
	defer pH.RUnlock()

	return pH.finalValue
}

// gets promise channel
func (pH *paxosHandler) getPromiseChannel() chan types.PaxosValue {
	pH.RLock()
	defer pH.RUnlock()

	return pH.promiseChannel
}

// gets value channel
func (pH *paxosHandler) getValueChannel() chan types.PaxosValue {
	pH.RLock()
	defer pH.RUnlock()

	return pH.valueChannel
}

// increments paxos ID by the number of peerss
func (pH *paxosHandler) nextID() {
	pH.Lock()
	defer pH.Unlock()

	pH.paxosID += pH.nPeers
}

// creates a prepare message with given source and init paramaters
func (pH *paxosHandler) createPrepareMessage(source string, value types.PaxosValue) types.PaxosPrepareMessage {
	pH.Lock()
	defer pH.Unlock()

	paxosPrepareMsg := types.PaxosPrepareMessage{
		Step:   pH.step,
		ID:     pH.paxosID,
		Source: source,
	}

	// init parameters with created value, phase 1 and 0 promises
	pH.proposedValue = value
	pH.currentPhase = paxosPhase1
	pH.promises = initValue

	return paxosPrepareMsg
}

// creates propose message with given value
func (pH *paxosHandler) createProposeMessage(value types.PaxosValue) types.PaxosProposeMessage {
	pH.RLock()
	defer pH.RUnlock()

	paxosProposeMsg := types.PaxosProposeMessage{
		Step:  pH.step,
		ID:    pH.paxosID,
		Value: value,
	}

	return paxosProposeMsg
}

// responds to prepare message by sending promise message if conditions are fulfilled
func (pH *paxosHandler) respondToPrepareMsg(msg types.PaxosPrepareMessage) *types.PaxosPromiseMessage {
	pH.Lock()
	defer pH.Unlock()

	// check step
	if pH.step != msg.Step {
		return nil
	}

	// check message ID (0 < ID < max ID)
	if msg.ID <= initValue || msg.ID <= pH.maxID {
		return nil
	}

	// set parameters
	pH.running = running
	pH.maxID = msg.ID

	// create promise message
	var acceptedValue *types.PaxosValue
	if (pH.acceptedValue != types.PaxosValue{}) {
		acceptedValue = &pH.acceptedValue
	}

	paxosPromiseMsg := types.PaxosPromiseMessage{
		Step:          pH.step,
		ID:            msg.ID,
		AcceptedID:    pH.acceptedID,
		AcceptedValue: acceptedValue,
	}

	return &paxosPromiseMsg
}

// responds to propose message by sending accept message if conditions are fulfilled
func (pH *paxosHandler) respondToProposeMsg(msg types.PaxosProposeMessage) *types.PaxosAcceptMessage {
	pH.Lock()
	defer pH.Unlock()

	// check step
	if pH.step != msg.Step {
		return nil
	}

	// check message ID (ID < max ID)
	if msg.ID != pH.maxID {
		return nil
	}

	// set parameters
	pH.running = running

	if (pH.acceptedValue == types.PaxosValue{}) {
		pH.acceptedID = msg.ID
		pH.acceptedValue = msg.Value
	}

	// create accept message
	paxosAcceptMsg := types.PaxosAcceptMessage{
		Step:  pH.step,
		ID:    msg.ID,
		Value: msg.Value,
	}

	return &paxosAcceptMsg
}

// responds to promise message by checking if threshold has been reached
func (pH *paxosHandler) respondToPromiseMsg(msg types.PaxosPromiseMessage) {
	pH.Lock()
	defer pH.Unlock()

	// check step
	if pH.step != msg.Step {
		return
	}

	// check current phase
	if pH.currentPhase != paxosPhase1 {
		return
	}

	// increment promises count
	pH.promises++

	// update best value if needed
	if msg.AcceptedID > pH.bestID {
		pH.bestID = msg.ID
		pH.bestValue = *msg.AcceptedValue
	}

	// if threshold is reached, init phase 2 and return good value to promise channel
	if pH.promises >= pH.threshold {
		pH.currentPhase = paxosPhase2
		if pH.bestID != 0 {
			pH.promiseChannel <- pH.bestValue
		} else {
			pH.promiseChannel <- pH.proposedValue
		}
	}
}

// responds to accept message by sending TLC message if conditions fulfilled
func (pH *paxosHandler) respondToAcceptMsg(msg types.PaxosAcceptMessage, n *node) (*types.TLCMessage, error) {
	pH.Lock()
	defer pH.Unlock()

	// check step
	if pH.step != msg.Step {
		return &types.TLCMessage{}, nil
	}

	// set parameters
	pH.running = running

	pH.acceptedValues[msg.Value]++

	// if threshold has been reached, consensus has been reached
	if pH.acceptedValues[msg.Value] >= pH.threshold {

		// build blockchain block
		block := pH.buildBlock(msg.Value, n.conf)

		// store block in store
		err := pH.storeBlock(block, n.conf)
		if err != nil {
			return nil, err
		}

		// create TLC message
		TLCMsg := types.TLCMessage{
			Step:  pH.step,
			Block: block,
		}

		// increment step
		pH.step++

		// check if consensus has been reached for future steps
		err = pH.catchUp(n.conf)

		// clear paxos instance
		pH.clearInstance(n.conf)

		return &TLCMsg, err
	}

	return &types.TLCMessage{}, nil
}

// responds to TLC message and check if broadcast is needed
func (pH *paxosHandler) respondToTLCMsg(msg types.TLCMessage, n *node) (bool, error) {
	pH.Lock()
	defer pH.Unlock()

	// check step
	if pH.step > msg.Step {
		return true, nil
	}

	// append message to TLC map with correspond step
	pH.TLCMap[msg.Step] = append(pH.TLCMap[msg.Step], msg)

	// if message step is the same as current step
	if msg.Step == pH.step {
		pH.running = running

		// if threshold has been reached, consensus has been reached
		if uint(len(pH.TLCMap[pH.step])) >= pH.threshold {
			// store block in store
			err := pH.storeBlock(msg.Block, n.conf)

			// set final value
			pH.finalValue = msg.Block.Value
			if err != nil {
				return true, err
			}

			// increment step
			pH.step++

			// check if consensus has been reached for future steps
			err = pH.catchUp(n.conf)

			// clear paxos instance
			pH.clearInstance(n.conf)

			return false, err
		}
	}

	return true, nil
}

// builds new blockchain block
func (pH *paxosHandler) buildBlock(paxosValue types.PaxosValue, conf peer.Configuration) types.BlockchainBlock {
	// set final value
	pH.finalValue = paxosValue
	store := conf.Storage.GetBlockchainStore()

	// get last blockchain block
	lastBlock := store.Get(storage.LastBlockKey)
	if lastBlock == nil {
		lastBlock = make([]byte, 32)
	}

	// compute data to hash
	data := []byte(strconv.Itoa(int(pH.step)))
	data = append(data, []byte(paxosValue.UniqID)...)
	data = append(data, []byte(paxosValue.Filename)...)
	data = append(data, []byte(paxosValue.Metahash)...)
	data = append(data, lastBlock...)

	// hash data
	h := crypto.SHA256.New()
	h.Write(data)
	hashSlice := h.Sum(nil)

	// create block
	block := types.BlockchainBlock{
		Index:    pH.step,
		Hash:     hashSlice,
		Value:    paxosValue,
		PrevHash: lastBlock,
	}

	return block
}

// stores block in store
func (pH *paxosHandler) storeBlock(block types.BlockchainBlock, conf peer.Configuration) error {
	bStore := conf.Storage.GetBlockchainStore()

	// compute key and block to store
	key := hex.EncodeToString(block.Hash)

	buf, err := block.Marshal()
	if err != nil {
		return err
	}

	// store block
	bStore.Set(key, buf)
	// store previous block
	bStore.Set(storage.LastBlockKey, block.Hash)

	// store filename and hash in naming store
	nStore := conf.Storage.GetNamingStore()
	nStore.Set(block.Value.Filename, []byte(block.Value.Metahash))

	return nil
}

// check if consensus for higher steps have been reached
func (pH *paxosHandler) catchUp(conf peer.Configuration) error {
	// if threshold has been reached, consensus has been reached
	for uint(len(pH.TLCMap[pH.step])) >= pH.threshold {
		TLCMsg := pH.TLCMap[pH.step][pH.threshold-1]

		// store block
		err := pH.storeBlock(TLCMsg.Block, conf)
		if err != nil {
			return err
		}
		// update step
		pH.step++
	}

	return nil
}
