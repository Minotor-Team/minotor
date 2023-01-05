package impl

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
)

// starts the node
func (n *node) Start() error {
	// indicate that the node is running
	atomic.StoreUint32(&n.run, 1)

	// start listening on incoming messages
	n.wg.Add(1)
	go n.listeningRoutine()

	if n.conf.AntiEntropyInterval > 0 {
		n.wg.Add(1)
		go n.antiEntropyMechanism()
	}

	if n.conf.HeartbeatInterval > 0 {
		n.wg.Add(1)
		go n.heartbeatMechanism()
	}

	return nil
}

// stops the node
func (n *node) Stop() error {
	// indicate that the node is not running anymore
	atomic.StoreUint32(&n.run, 0)
	close(n.stopChannel)
	// wait for all routines to be finished
	n.acksHandler.close()
	n.requestsHandler.close()
	n.searchsHandler.close()
	n.wg.Wait()
	return nil
}

// listens to incoming messages as long as the node is running
func (n *node) listeningRoutine() {
	defer n.wg.Done()

	for atomic.LoadUint32(&n.run) == 1 {
		// receive the packet
		pkt, err := n.soc.Recv(time.Second * 1)

		// if error different from timeout, stop the node
		if errors.Is(err, transport.TimeoutError(0)) {
			continue
		} else if err != nil {
			atomic.StoreUint32(&n.run, 0)
			log.Err(err).Msg("failed to receive packet")
			break
		}

		// process packet if destination is the node or relay it to the good destination
		dest := pkt.Header.Destination
		myAddr := n.soc.GetAddress()

		if dest == myAddr {
			err = n.reg.ProcessPacket(pkt)
		} else {
			pkt.Header.RelayedBy = myAddr
			err = n.soc.Send(dest, pkt, time.Second*1)
		}

		// display error if any
		if err != nil {
			atomic.StoreUint32(&n.run, 0)
			log.Err(err).Msg("failed to process packet")
		}
	}

}
