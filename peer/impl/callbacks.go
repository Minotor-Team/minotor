package impl

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// reputation
func (n *node) ExecLikeMessage(msg types.Message, pkt transport.Packet) error {
	fmt.Println("Like recu")
	likeMsg, conv := msg.(*types.LikeMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}
	fmt.Println(likeMsg)
	fmt.Println("likeMSG " + pkt.String())
	return nil
}

func (n *node) ExecDislikeMessage(msg types.Message, pkt transport.Packet) error {
	// TODO implement
	fmt.Println("Dislike recu")
	dislikeMsg, conv := msg.(*types.DislikeMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}
	fmt.Println("dislikeMSG " + dislikeMsg.HTML())
	fmt.Println("dislikeMSG " + pkt.String())
	return nil
}

// processes chat message
func (n *node) ExecChatMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to ChatMessage
	chatMsg, conv := msg.(*types.ChatMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	fmt.Println("Source " + pkt.Header.Source)
	fmt.Println(chatMsg)

	// log message
	log.Info().Msgf("%s", chatMsg)

	return nil
}

// processes rumors message locally
func (n *node) processRumorsMsgLocally(msg types.Message, pkt transport.Packet) error {
	// convert message to RumorsMessage
	rumorsMsg, conv := msg.(*types.RumorsMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// iterate on rumors to process it one by one
	for _, r := range rumorsMsg.Rumors {
		err := processRumor(n, r, &pkt)
		if err != nil {
			return err
		}
	}
	return nil
}

// processes rumors message
func (n *node) ExecRumorsMessage(msg types.Message, pkt transport.Packet) error {

	// if source equals destination, process message locally
	if pkt.Header.Source == pkt.Header.Destination {
		return n.processRumorsMsgLocally(msg, pkt)
	}

	// convert message to RumorsMessage
	rumorsMsg, conv := msg.(*types.RumorsMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process rumors
	discoveredRumor, err := n.rumorsHandler.processRumors(n, rumorsMsg, &pkt)
	if err != nil {
		return err
	}

	// send an ack back to remote node
	err = n.sendBackAck(pkt)
	if err != nil {
		return err
	}

	// if new rumor, send it to a random neighbour
	if discoveredRumor {
		neighbours := n.routingTable.getNeighbours()
		sentRumorNeighbours := make(map[string]bool)
		sentRumorNeighbours[pkt.Header.Source] = true
		sentRumorNeighbours[pkt.Header.RelayedBy] = true
		pkt.Header.RelayedBy = n.soc.GetAddress()
		n.searchNeighbourAndSndPkt(pkt, neighbours, sentRumorNeighbours)
	}

	return err
}

// sends an ack back to remote node
func (n *node) sendBackAck(pkt transport.Packet) error {
	myAddr := n.soc.GetAddress()

	// create ack header and header
	ackHeader := transport.NewHeader(myAddr, myAddr, pkt.Header.Source, 0)
	ack := types.Message(&types.AckMessage{
		AckedPacketID: pkt.Header.PacketID,
		Status:        types.StatusMessage(n.rumorsHandler.getSequenceMap()),
	})

	// create packet and send it
	err := n.createPktAndSend(&ackHeader, ack, ackHeader.Destination)

	return err
}

// processes empty message
func (n *node) ExecEmptyMessage(types.Message, transport.Packet) error { return nil }

// processes status message
func (n *node) ExecStatusMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to StatusMessage
	statusMsg, conv := msg.(*types.StatusMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	myAddr := n.soc.GetAddress()

	// 1. check if remote node has Rumors that local node doesn’t have
	localMissingRumors := n.rumorsHandler.isLocalMissingRumors(sequenceMap(*statusMsg))
	if localMissingRumors {
		// send status message to remote node with sequence map
		statusHeader := transport.NewHeader(myAddr, myAddr, pkt.Header.Source, 0)
		status := types.StatusMessage(n.rumorsHandler.getSequenceMap())
		err := n.createPktAndSend(&statusHeader, status, statusHeader.Destination)
		if err != nil {
			return err
		}
	}

	// 2. check if local node has Rumors that the remote node doesn’t have
	remoteMissingRumors := n.rumorsHandler.getRemoteMissingRumors(sequenceMap(*statusMsg))
	if len(remoteMissingRumors) > 0 {
		// send missing Rumors to remote node
		rumorsHeader := transport.NewHeader(myAddr, myAddr, pkt.Header.Source, 0)
		rumors := types.RumorsMessage{Rumors: remoteMissingRumors}
		err := n.createPktAndSend(&rumorsHeader, rumors, rumorsHeader.Destination)
		if err != nil {
			return err
		}
	}

	// 4. if both node have the same rumors and with certain probability (ContinueMongering system)
	if !localMissingRumors && len(remoteMissingRumors) == 0 && rand.Float64() <= n.conf.ContinueMongering {
		// send status message to random node different from source
		neighbours := n.routingTable.getNeighbours()
		sentRumorNeighbours := make(map[string]bool)
		sentRumorNeighbours[pkt.Header.Source] = true
		statusHeader := transport.NewHeader(myAddr, myAddr, pkt.Header.Source, 0)
		status := types.StatusMessage(n.rumorsHandler.getSequenceMap())
		statusPkt, err := n.createPkt(&statusHeader, status)
		if err != nil {
			return err
		}
		n.searchNeighbourAndSndPkt(statusPkt, neighbours, sentRumorNeighbours)
	}

	return nil
}

// processes ack message
func (n *node) ExecAckMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to AckMessage
	ackMsg, conv := msg.(*types.AckMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// stop waiting for ack message by deleting ack channel
	n.acksHandler.deleteChannel(pkt.Header.PacketID)

	// process status message contained in AckMessage
	return n.ExecStatusMessage(&ackMsg.Status, pkt)
}

// processes private message
func (n *node) ExecPrivateMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to PrivateMessage
	privateMsg, conv := msg.(*types.PrivateMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	myAddr := n.soc.GetAddress()

	// if node socket address is in recipients list, process message,
	// otherwise do nothing, message is not for this node
	_, inRecipients := privateMsg.Recipients[myAddr]

	if inRecipients {
		privatePacket := transport.Packet{
			Header: pkt.Header,
			Msg:    privateMsg.Msg,
		}
		return n.reg.ProcessPacket(privatePacket)
	}

	return nil
}

// processes data reply message
func (n *node) ExecDataReplyMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to DataReplyMessage
	dataReplyMsg, conv := msg.(*types.DataReplyMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// store message pair key/value
	if dataReplyMsg.Value != nil {
		store := n.conf.Storage.GetDataBlobStore()
		store.Set(dataReplyMsg.Key, dataReplyMsg.Value)
	}

	// delete channel created in Download function
	n.requestsHandler.deleteChannel(dataReplyMsg.RequestID)
	return nil
}

// processes data request message
func (n *node) ExecDataRequestMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to DataRequestMessage
	dataRequestMsg, conv := msg.(*types.DataRequestMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// get metafile from store using given key
	store := n.conf.Storage.GetDataBlobStore()
	metafile := store.Get(dataRequestMsg.Key)

	dataReplyMessage := types.DataReplyMessage{
		RequestID: dataRequestMsg.RequestID,
		Key:       dataRequestMsg.Key,
		Value:     metafile,
	}

	repMsg, err := n.reg.MarshalMessage(dataReplyMessage)
	if err != nil {
		return err
	}

	// send data to requesting peer
	return n.Unicast(pkt.Header.Source, repMsg)
}

// processes search reply message
func (n *node) ExecSearchReplyMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to SearchReplyMessage
	searchReplyMsg, conv := msg.(*types.SearchReplyMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// iterate on message responses
	for _, r := range searchReplyMsg.Responses {
		// update store and catalogue
		n.updateStoreAndCatalog(searchReplyMsg, r, pkt)

		for _, chunk := range r.Chunks {
			if chunk != nil {
				n.catalog.addEntry(string(chunk), pkt.Header.Source)
			}
		}
	}

	// check if complete file is available and trigger channel in SearchFirst function
	n.checkAvailabilityAndTrigger(searchReplyMsg, pkt)

	return nil
}

// updates store and catalogue with message content
func (n *node) updateStoreAndCatalog(sRM *types.SearchReplyMessage, res types.FileInfo, pkt transport.Packet) {
	store := n.conf.Storage.GetNamingStore()
	n.store.addEntry(sRM.RequestID, res.Name)
	n.catalog.addEntry(res.Metahash, pkt.Header.Source)
	store.Set(res.Name, []byte(res.Metahash))
}

// checks if complete file is available and trigger channel in SearchFirst function
func (n *node) checkAvailabilityAndTrigger(searchReplyMsg *types.SearchReplyMessage, pkt transport.Packet) {
	_, inTable := n.searchsHandler.getChannel(searchReplyMsg.RequestID)
	if inTable {
		entireFileIsAvailable := false
		filename := ""

		// iterate on responses to update store and catalogue and check if full file available
		for _, r := range searchReplyMsg.Responses {
			n.updateStoreAndCatalog(searchReplyMsg, r, pkt)

			noMissingChunk := true
			// iterate on chunks to check if one is missing
			for _, chunk := range r.Chunks {
				if chunk != nil {
					n.catalog.addEntry(string(chunk), pkt.Header.Source)
				} else {
					noMissingChunk = false
					break
				}
			}

			// if no missing chuck, it means that complete file is available and return filename
			if noMissingChunk && !entireFileIsAvailable {
				filename = r.Name
				entireFileIsAvailable = true
			}
		}

		// trigger channel in SearchFirst function is file available
		if entireFileIsAvailable {
			n.searchsHandler.triggerChannel(searchReplyMsg.RequestID, filename)
		}
	}
}

// processes search request message
func (n *node) ExecSearchRequestMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to SearchRequestMessage
	searchRequestMsg, conv := msg.(*types.SearchRequestMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// update budget and create pattern
	budget := searchRequestMsg.Budget - 1
	requestID := searchRequestMsg.RequestID
	neighbours := []string{}
	filenames := []string{}
	metahashes := [][]byte{}
	reg := *regexp.MustCompile(searchRequestMsg.Pattern)

	// get filenames and metahashes from store
	store := n.conf.Storage.GetNamingStore()
	store.ForEach(func(key string, val []byte) bool {
		if reg.MatchString(key) {
			if n.conf.Storage.GetDataBlobStore().Get(string(val)) != nil {
				filenames = append(filenames, key)
				metahashes = append(metahashes, val)
			}
		}
		return true
	})

	// send reply message to requesting peer with requested data
	err := n.sendReplyMsg(*searchRequestMsg, pkt.Header.RelayedBy, filenames, metahashes)
	if err != nil {
		return err
	}

	for _, addr := range n.routingTable.getNeighbours() {
		if addr != pkt.Header.Source && addr != searchRequestMsg.Origin {
			neighbours = append(neighbours, addr)
		}
	}

	// forward search with available budget
	return n.sendSearchReq(neighbours, requestID, budget, reg, searchRequestMsg.Origin)
}

// sends replay with given filenames and metahashes
func (n *node) sendReplyMsg(sRM types.SearchRequestMessage, dest string, f []string, m [][]byte) error {
	responses := []types.FileInfo{}
	myAddr := n.soc.GetAddress()
	store := n.conf.Storage.GetDataBlobStore()

	// iterate on filename to retrieve data
	for i, filename := range f {

		metafile := store.Get(string(m[i]))
		chunkList := strings.Split(string(metafile), peer.MetafileSep)

		chunks := make([][]byte, len(chunkList))
		for i, chunkHexEncoded := range chunkList {
			chunk := store.Get(chunkHexEncoded)
			if chunk != nil {
				chunks[i] = []byte(chunkHexEncoded)
			}
		}

		response := types.FileInfo{
			Name:     filename,
			Metahash: string(m[i]),
			Chunks:   chunks,
		}
		responses = append(responses, response)
	}

	searchReplyHeader := transport.NewHeader(myAddr, myAddr, sRM.Origin, 0)
	searchReply := types.SearchReplyMessage{
		RequestID: sRM.RequestID,
		Responses: responses,
	}

	// send reply with responses
	return n.createPktAndSend(&searchReplyHeader, searchReply, dest)
}

// processes paxos prepare message
func (n *node) ExecPaxosPrepareMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to PaxosPrepareMessage
	paxosPrepareMsg, conv := msg.(*types.PaxosPrepareMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process prepare message and create promise message
	promise := n.paxosHandler.respondToPrepareMsg(*paxosPrepareMsg)
	if promise == nil {
		return nil
	}

	transportPromiseMsg, err := n.reg.MarshalMessage(promise)
	if err != nil {
		return err
	}

	// embed promise message in private message
	dests := map[string]struct{}{paxosPrepareMsg.Source: {}}
	privateMsg := types.PrivateMessage{
		Recipients: dests,
		Msg:        &transportPromiseMsg,
	}

	transportPrivateMsg, err := n.reg.MarshalMessage(privateMsg)
	if err != nil {
		return err
	}

	// broadcast private message
	err = n.Broadcast(transportPrivateMsg)
	return err
}

// processes paxos propose message
func (n *node) ExecPaxosProposeMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to PaxosProposeMessage
	paxosProposeMsg, conv := msg.(*types.PaxosProposeMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process propose message and create accept message
	accept := n.paxosHandler.respondToProposeMsg(*paxosProposeMsg)
	if accept == nil {
		return nil
	}

	transportAcceptMsg, err := n.reg.MarshalMessage(accept)
	if err != nil {
		return err
	}

	// broadcast accept message
	err = n.Broadcast(transportAcceptMsg)
	return err
}

// processes paxos promise message
func (n *node) ExecPaxosPromiseMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to PaxosPromiseMessage
	paxosPromiseMsg, conv := msg.(*types.PaxosPromiseMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process promise message
	n.paxosHandler.respondToPromiseMsg(*paxosPromiseMsg)

	return nil
}

// processes paxos accept message
func (n *node) ExecPaxosAcceptMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to PaxosAcceptMessage
	paxosAcceptMsg, conv := msg.(*types.PaxosAcceptMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process accept message and create TLC message
	TLCMsg, err := n.paxosHandler.respondToAcceptMsg(*paxosAcceptMsg, n)
	if err != nil || TLCMsg.Block.Hash == nil {
		return err
	}

	// broadcast TLC message
	err = n.broadcastTLC(TLCMsg)
	return err
}

// processes TLC message
func (n *node) ExecTLCMessage(msg types.Message, pkt transport.Packet) error {
	// convert message to TLCMessage
	TLCMsg, conv := msg.(*types.TLCMessage)
	if !conv {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	// process TLC message and check if need to be broadcasted
	broadcasted, err := n.paxosHandler.respondToTLCMsg(*TLCMsg, n)
	if err != nil {
		return err
	}

	// broadcast TLCMessage if not already done
	if !broadcasted {
		err := n.broadcastTLC(TLCMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

// broadcasts TLC message
func (n *node) broadcastTLC(TLCMsg *types.TLCMessage) error {
	// marshal message
	transportTLCMsg, err := n.reg.MarshalMessage(TLCMsg)
	if err != nil {
		return err
	}

	// broadcast message
	err = n.Broadcast(transportTLCMsg)

	return err
}
