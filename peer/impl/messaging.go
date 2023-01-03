package impl

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// reputation
// send like to a given destination
func (n *node) SendLike(dest string, msg transport.Message) error {
	addr, inTable := n.routingTable.getAddr(dest)

	if !inTable {
		return xerrors.Errorf("forwarding address not found: %v", dest)
	}

	myAddr := n.soc.GetAddress()

	// create packet
	header := transport.NewHeader(myAddr, myAddr, dest, 1)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}

	// send packet
	fmt.Println(("unicast addr:" + addr))
	fmt.Println((msg.Payload))
	err := n.soc.Send(addr, pkt, 0)

	return err
}

// send dislike to a given destination
func (n *node) SendDisLike(dest string, msg transport.Message) error {
	addr, inTable := n.routingTable.getAddr(dest)

	if !inTable {
		return xerrors.Errorf("forwarding address not found: %v", dest)
	}

	myAddr := n.soc.GetAddress()

	// create packet
	header := transport.NewHeader(myAddr, myAddr, dest, 1)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}

	// send packet
	err := n.soc.Send(addr, pkt, 0)

	return err
}

// sends a packet to a given destination
func (n *node) Unicast(dest string, msg transport.Message) error {
	// retrieve forwarding address
	addr, inTable := n.routingTable.getAddr(dest)

	if !inTable {
		return xerrors.Errorf("forwarding address not found: %v", dest)
	}

	myAddr := n.soc.GetAddress()

	// create packet
	header := transport.NewHeader(myAddr, myAddr, dest, 1)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}

	// send packet
	fmt.Println(("unicast addr:" + addr))
	fmt.Println(msg.Payload)
	err := n.soc.Send(addr, pkt, 0)

	return err
}

// broadcasts given message
func (n *node) Broadcast(msg transport.Message) error {
	fmt.Println(("real broad"))

	myAddr := n.soc.GetAddress()

	// create rumors header, message and then packet
	rumorsHeader := transport.NewHeader(myAddr, myAddr, "", 0)
	rumorsMsg := n.rumorsHandler.createRumorsMsg(myAddr, &msg)

	pkt, err := n.createPkt(&rumorsHeader, rumorsMsg)
	if err != nil {
		return err
	}

	neighbours := n.routingTable.getNeighbours()
	// select random neighbour if any, send rumor & wait for ack
	if len(neighbours) > 0 {
		sentRumorNeighbours := make(map[string]bool)
		n.searchNeighbourAndSndPkt(pkt, neighbours, sentRumorNeighbours)
		n.wg.Add(1)
		go n.waitAck(pkt, neighbours, sentRumorNeighbours)
	}

	// process rumor locally
	err = n.processRumorsMsgLocally(rumorsMsg, pkt)
	if err != nil {
		return err
	}

	return nil
}

// adds new known addresses to the node if different from node address
func (n *node) AddPeer(addr ...string) {
	n.routingTable.addEntries(addr)
}

// returns a copy of the node's routing table
func (n *node) GetRoutingTable() peer.RoutingTable {
	return n.routingTable.getRoutingTable()
}

// sets the routing entry
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	if len(relayAddr) > 0 {
		n.routingTable.addEntry(origin, relayAddr)
	} else {
		n.routingTable.delete(origin)
	}
}

// waits to receive an ack and send pkt to another neighbours if timeout is reached
func (n *node) waitAck(pkt transport.Packet, neighbours []string, sentRumorNeighbours map[string]bool) {
	defer n.wg.Done()

	// create ack channel to trigger if ack received
	n.acksHandler.createChannel(pkt.Header.PacketID)
	ackChannel, inTable := n.acksHandler.getChannel(pkt.Header.PacketID)
	if !inTable {
		log.Err(xerrors.Errorf("channel not found for Packet ID: %v", pkt.Header.PacketID)).Msg("channel not found")
	}
	ackReceived := false

	// wait time out before checking if ack has been received
	if n.conf.AckTimeout != 0 {
		time.Sleep(n.conf.AckTimeout)
	} else {
		time.Sleep(100 * time.Millisecond)
	}

	// while ack is not received
	for !ackReceived {
		select {
		// stop if ack is received
		case <-*ackChannel:
			ackReceived = true
		// choose another random neighbour to whom the packet has not yet been sent, send it and wait
		default:
			if n.conf.AckTimeout == 0 {
				time.Sleep(100 * time.Millisecond)
				break
			}

			ret := n.searchNeighbourAndSndPkt(pkt, neighbours, sentRumorNeighbours)
			if ret {
				return
			}

			time.Sleep(n.conf.AckTimeout)
		}
	}
}

// searchs for random neighbour and send packet
func (n *node) searchNeighbourAndSndPkt(pkt transport.Packet, neighbours []string, rMap map[string]bool) bool {

	neighbourAddress := ""

	// choose a random neighbour to whom the packet has not yet been sent
	for _, n := range neighbours {
		if !rMap[n] {
			neighbourAddress = n
			rMap[n] = true
			break
		}
	}

	if neighbourAddress == "" {
		return true
	}

	// send packet to chosen neighbour if any
	pkt.Header.Destination = neighbourAddress
	err := n.soc.Send(pkt.Header.Destination, pkt, 0)
	if err != nil {
		log.Err(err).Msg("failed to send message")
		return true
	}

	return false
}

// returns a transport packet from header and message
func (n *node) createPkt(header *transport.Header, msg types.Message) (transport.Packet, error) {

	// transforms the message so that it is ready to be sent
	transportMsg, err := n.reg.MarshalMessage(msg)
	if err != nil {
		return transport.Packet{}, xerrors.Errorf("failed to marshal message")
	}

	// create transport packet
	pkt := transport.Packet{
		Header: header,
		Msg:    &transportMsg,
	}

	return pkt, nil

}

// creates transport packet from header and message and send it
func (n *node) createPktAndSend(header *transport.Header, msg types.Message, dest string) error {

	// create transport packet
	pkt, err := n.createPkt(header, msg)
	if err != nil {
		return err
	}

	// send packet
	err = n.soc.Send(dest, pkt, 0)
	if err != nil {
		return xerrors.Errorf("failed to send packet")
	}

	return nil
}
