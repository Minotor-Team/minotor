package impl

import (
	"sync"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
)

func newPaxosLikeHandler(conf peer.Configuration) *paxosLikeHandler {
	return &paxosLikeHandler{
		running:         idle,
		step:            initValue,
		stepUpdate:      newStepUpdate(),
		maxID:           initValue,
		bestID:          initValue,
		acceptedID:      initValue,
		promises:        initValue,
		currentPhase:    paxosPhase1,
		paxosID:         conf.PaxosID,
		nPeers:          conf.TotalPeers,
		threshold:       uint(conf.PaxosThreshold(conf.TotalPeers)),
		bestValue:       types.PaxosLike{},
		acceptedValue:   types.PaxosLike{},
		proposedValue:   types.PaxosLike{},
		finalValue:      types.PaxosLike{},
		acceptedValues:  make(map[types.PaxosLike]uint),
		promiseChannel:  make(chan types.PaxosLike, 1),
		valueChannel:    make(chan types.PaxosLike, 1),
		instanceChannel: make(chan struct{}),
	}
}

type paxosLikeHandler struct {
	sync.RWMutex
	running bool
	step    uint
	// keep in memory if the map was updated for the current step
	stepUpdate      *stepUpdate
	maxID           uint
	bestID          uint
	acceptedID      uint
	promises        uint
	currentPhase    uint
	paxosID         uint
	nPeers          uint
	threshold       uint
	bestValue       types.PaxosLike
	acceptedValue   types.PaxosLike
	proposedValue   types.PaxosLike
	finalValue      types.PaxosLike
	acceptedValues  map[types.PaxosLike]uint
	promiseChannel  chan types.PaxosLike
	valueChannel    chan types.PaxosLike
	instanceChannel chan struct{}
}

type stepUpdate struct {
	sync.RWMutex
	stepUpdate map[uint]bool
}

func newStepUpdate() *stepUpdate {
	return &stepUpdate{
		stepUpdate: make(map[uint]bool),
	}
}

func (stepUpd *stepUpdate) setStepUpdate(step uint, b bool) {
	stepUpd.Lock()
	defer stepUpd.Unlock()
	stepUpd.stepUpdate[step] = b
}

func (pH *paxosLikeHandler) initPaxosInstance() (bool, chan struct{}) {
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

func (pH *paxosLikeHandler) getStep() uint {
	pH.RLock()
	defer pH.RUnlock()

	return pH.step
}

func (pH *paxosLikeHandler) getValueChannel() chan types.PaxosLike {
	pH.RLock()
	defer pH.RUnlock()

	return pH.valueChannel
}

func (pH *paxosLikeHandler) clearInstance(conf peer.Configuration) {
	pH.running = idle
	pH.maxID = initValue
	pH.bestID = initValue
	pH.acceptedID = initValue
	pH.promises = initValue
	pH.currentPhase = paxosPhase1
	pH.paxosID = conf.PaxosID
	pH.nPeers = conf.TotalPeers
	pH.threshold = uint(conf.PaxosThreshold(conf.TotalPeers))
	pH.bestValue = types.PaxosLike{}
	pH.acceptedValue = types.PaxosLike{}
	pH.proposedValue = types.PaxosLike{}
	pH.acceptedValues = make(map[types.PaxosLike]uint)
	pH.promiseChannel = make(chan types.PaxosLike, 1)
	pH.valueChannel <- pH.finalValue
	close(pH.valueChannel)
	pH.valueChannel = make(chan types.PaxosLike, 1)
	close(pH.instanceChannel)
	pH.instanceChannel = make(chan struct{})
}
