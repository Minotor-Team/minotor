package impl

import (
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// implements anti entroypy mechanism
func (n *node) antiEntropyMechanism() {
	defer n.wg.Done()

	// create ticker with given interval
	ticker := time.NewTicker(n.conf.AntiEntropyInterval)
	for {
		select {
		// if ticker is triggered, do routine
		case <-ticker.C:
			neighbours := n.routingTable.getNeighbours()
			if len(neighbours) > 0 {
				myAddr := n.soc.GetAddress()

				// send status message to random neighbour
				neighbourAddress := neighbours[0]
				statusHeader := transport.NewHeader(myAddr, myAddr, neighbourAddress, 0)
				status := types.StatusMessage(n.rumorsHandler.getSequenceMap())

				err := n.createPktAndSend(&statusHeader, &status, statusHeader.Destination)
				if err != nil {
					log.Err(err).Msg("failed to send status message")
					return
				}
			}
		// if stop function is called, stop mechanism
		case <-n.stopChannel:
			return
		}
	}
}

// implements heart beat mechanism
func (n *node) heartbeatMechanism() {
	defer n.wg.Done()

	// create ticker with small timeout to trigger fast
	ticker := time.NewTicker(1 * time.Microsecond)
	for {
		select {
		// if ticker is triggered, do routine
		case <-ticker.C:
			// broadcast empty message
			emptyMsg, err := n.reg.MarshalMessage(types.EmptyMessage{})
			if err != nil {
				log.Err(err).Msg("failed to marshal empty message")
				return
			}

			err = n.Broadcast(emptyMsg)
			if err != nil {
				log.Err(err).Msg("failed to broadcast empty message")
				return
			}
			// update ticker with given interval
			ticker = time.NewTicker(n.conf.HeartbeatInterval)
		// if stop function is called, stop mechanism
		case <-n.stopChannel:
			return
		}

	}
}
