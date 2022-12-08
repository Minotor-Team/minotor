package impl

import (
	"sync"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// initialises rumors handler
func newRumorsHandler() *rumorsHandler {
	return &rumorsHandler{
		sequenceMap: make(sequenceMap),
		rumors:      make(rumors),
	}
}

type rumorsHandler struct {
	sync.RWMutex
	sequenceMap sequenceMap
	rumors      rumors
}

type sequenceMap map[string]uint
type rumors map[string][]types.Rumor

// creates RumorsMessage
func (rH *rumorsHandler) createRumorsMsg(origin string, msg *transport.Message) types.Message {
	rH.Lock()
	defer rH.Unlock()

	// update sequence map
	sequence := rH.sequenceMap[origin] + 1
	rH.sequenceMap[origin] = sequence

	// create rumor packet and add it to node rumors map
	rumor := types.Rumor{
		Origin:   origin,
		Sequence: sequence,
		Msg:      msg,
	}

	rH.rumors[origin] = append(rH.rumors[origin], rumor)

	// convert rumor to message type
	rumorsMsg := types.Message(&types.RumorsMessage{
		Rumors: []types.Rumor{rumor},
	})

	return rumorsMsg
}

// returns a copy of the sequence map
func (rH *rumorsHandler) getSequenceMap() sequenceMap {
	rH.RLock()
	defer rH.RUnlock()

	// initialize table to be returned
	copySM := make(sequenceMap)

	// copy sequence map
	for k, v := range rH.sequenceMap {
		copySM[k] = v
	}

	return copySM
}

// processes RumorsMessage
func (rH *rumorsHandler) processRumors(n *node, r *types.RumorsMessage, pkt *transport.Packet) (bool, error) {
	rH.Lock()
	defer rH.Unlock()
	processedRumor := false

	// iterates on rumors
	for _, m := range r.Rumors {
		lastRumorSequence := rH.sequenceMap[m.Origin]

		if m.Sequence == lastRumorSequence+1 {

			// update routing table, sequence map and rumors map
			processedRumor = true
			n.routingTable.updateRT(m.Origin, pkt.Header.RelayedBy)
			rH.sequenceMap[m.Origin] = m.Sequence
			rH.rumors[m.Origin] = append(rH.rumors[m.Origin], m)

			// process rumor
			rH.Unlock()
			err := processRumor(n, m, pkt)
			rH.Lock()
			if err != nil {
				return false, err
			}
		}
	}

	return processedRumor, nil
}

// processes rumor
func processRumor(n *node, rumor types.Rumor, pkt *transport.Packet) error {

	// create rumor packet
	rumorPkt := transport.Packet{
		Header: pkt.Header,
		Msg:    rumor.Msg,
	}

	// process rumor packet
	err := n.reg.ProcessPacket(rumorPkt)

	return err
}

// checks if sequence map is up-to-date by comparing it with remote sequence map
func (rH *rumorsHandler) isLocalMissingRumors(sequenceMap sequenceMap) bool {
	rH.RLock()
	defer rH.RUnlock()

	// iterate on remote sequence map and check if all sequences exists in local sequence map
	for a, s := range sequenceMap {
		sequence, inMap := rH.sequenceMap[a]
		if !inMap || s > sequence {
			return true
		}
	}

	return false
}

// returns rumors that are missing on remote node
func (rH *rumorsHandler) getRemoteMissingRumors(sequenceMap sequenceMap) []types.Rumor {
	rH.RLock()
	defer rH.RUnlock()

	// iterate on rumors and check that sequence maps are the same
	// or return missing ones otherwise
	for a, r := range rH.rumors {
		if sequenceMap[a] < rH.sequenceMap[a] {
			return r[sequenceMap[a]:]
		}
	}

	return nil
}
