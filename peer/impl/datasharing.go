package impl

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// uploads come given data
func (n *node) Upload(data io.Reader) (string, error) {
	var metafile string
	var chunkList []byte

	store := n.conf.Storage.GetDataBlobStore()

	// read data and create metafile and chunks
	for {
		buff := make([]byte, n.conf.ChunkSize)
		n, err := data.Read(buff)

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		chunk := buff[:n]

		h := crypto.SHA256.New()
		h.Write(chunk)
		chunkSlice := h.Sum(nil)
		chunkHexEncoded := hex.EncodeToString(chunkSlice)

		chunkList = append(chunkList, chunkSlice...)
		store.Set(chunkHexEncoded, chunk)
		metafile += chunkHexEncoded + peer.MetafileSep
	}

	metafile = metafile[:len(metafile)-1]

	// encrypt metafile
	h := crypto.SHA256.New()
	h.Write(chunkList)
	metahashSlice := h.Sum(nil)
	metahashHex := hex.EncodeToString(metahashSlice)
	store.Set(metahashHex, []byte(metafile))

	return metahashHex, nil
}

// downloads data from given metahash
func (n *node) Download(metahash string) ([]byte, error) {
	var err error
	// search file locally
	store := n.conf.Storage.GetDataBlobStore()
	metafile := store.Get(metahash)
	var data []byte

	// request metafile to random remote peers with data
	if metafile == nil {
		metafile, err = n.getDataFromRemotePeer(metahash, store)
		if err != nil {
			return nil, err
		}
	}

	chunkHexEncoded := strings.Fields(string(metafile))

	// retrieve chunks locally or remotly if not available
	for _, chunkSlice := range chunkHexEncoded {
		chunk := store.Get(chunkSlice)
		if chunk == nil {
			chunk, err = n.getDataFromRemotePeer(chunkSlice, store)
			if err != nil {
				return nil, err
			}
		}
		data = append(data, chunk...)
	}

	return data, nil
}

// maps name to given metahash
func (n *node) Tag(name string, mh string) error {
	store := n.conf.Storage.GetNamingStore()

	// check if name not already in the naming store
	storedName := store.Get(name)
	if storedName != nil {
		return xerrors.Errorf("already existing name : %v", name)
	}

	// if only one node set name
	if n.conf.TotalPeers <= 1 {
		store.Set(name, []byte(mh))
		return nil
	}

	// wait for paxos instance to finish
	instanceRunning, instanceChannel := n.paxosHandler.initPaxosInstance()
	for instanceRunning {
		select {
		case <-instanceChannel:
			instanceRunning, instanceChannel = n.paxosHandler.initPaxosInstance()
		case <-n.stopChannel:
			return nil
		}
	}

	// loop on the paxos instance until consensus is finished
	startingStep := n.paxosHandler.getStep()
	for {
		step := n.paxosHandler.getStep()
		// if step different from initial one, restart tag if value stored is not ours
		if startingStep != step {
			storedName = store.Get(name)
			if storedName != nil {
				return nil
			}
			return n.Tag(name, mh)
		}

		// launch paxos phase 1
		proposedValue, err := n.PaxosPhase1(name, mh)
		if (err != nil || *proposedValue == types.PaxosValue{}) {
			return err
		}

		// launch paxos phase 2
		ret, err := n.PaxosPhase2(*proposedValue, name, mh, store)
		if err != nil || ret {
			return err
		}
	}
}

// starts paxos phase 1
func (n *node) PaxosPhase1(name string, mh string) (*types.PaxosValue, error) {
	for {
		myAddr := n.soc.GetAddress()

		// broadcast pepare message with proposed value
		proposedValue := types.PaxosValue{
			UniqID:   xid.New().String(),
			Filename: name,
			Metahash: mh,
		}

		paxosPrepareMsg := n.paxosHandler.createPrepareMessage(myAddr, proposedValue)
		promiseChannel := n.paxosHandler.getPromiseChannel()

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
			n.paxosHandler.nextID()
		// if stop function is called, stop function
		case <-n.stopChannel:
			return &types.PaxosValue{}, nil
		}
	}
}

// starts paxos phase 2
func (n *node) PaxosPhase2(value types.PaxosValue, name string, mh string, store storage.Store) (bool, error) {
	// broadcast propose message with given value
	paxosProposeMsg := n.paxosHandler.createProposeMessage(value)
	valueChannel := n.paxosHandler.getValueChannel()

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
		finalValue := n.paxosHandler.getFinalValue()
		if (finalValue != types.PaxosValue{}) {
			if finalValue.Filename == name && finalValue.Metahash == mh {
				return true, nil
			}
			return true, n.Tag(name, mh)
		}
		// increment paxos ID and restart loop
		n.paxosHandler.nextID()
	// if stop function is called, stop function
	case <-n.stopChannel:
		return true, nil
	}
	return false, nil
}

// resolves given name to metahash
func (n *node) Resolve(name string) string {
	store := n.conf.Storage.GetNamingStore()
	return string(store.Get(name))
}

// returns catalog
func (n *node) GetCatalog() peer.Catalog {
	return n.catalog.getCatalog()
}

// update catalog with given key/peer
func (n *node) UpdateCatalog(key string, peer string) {
	n.catalog.addEntry(key, peer)
}

// returns list of filenames from local store or neighbours
func (n *node) SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) ([]string, error) {
	requestID := xid.New().String()
	filenames := []string{}

	// create request entry in store
	n.store.createRequestMap(requestID)
	// send search request with given budget and requestID
	err := n.sendSearchReq(n.routingTable.getNeighbours(), requestID, budget, reg, n.soc.GetAddress())
	if err != nil {
		return filenames, err
	}

	// wait for timeout
	time.Sleep(timeout)

	// retrieve filenames locally and remotly
	filenames = n.retrieveFilenames(requestID, reg, filenames)

	return filenames, nil
}

// search first peer (local or remote) that has entire file available
func (n *node) SearchFirst(pattern regexp.Regexp, conf peer.ExpandingRing) (string, error) {
	// check if file is available locally
	local := n.localFullyKnowsFile(pattern)
	if local != "" {
		return local, nil
	}

	requestID := xid.New().String()
	trials := uint(1)
	budget := conf.Initial

	// create channel with requestID
	n.searchsHandler.createChannel(requestID)
	searchChannel, inTable := n.searchsHandler.getChannel(requestID)
	if !inTable {
		return "", xerrors.Errorf("channel not found for requestID: %v", requestID)
	}
	defer n.searchsHandler.deleteChannel(requestID)

	// send search request to neighbours with given budget
	err := n.sendSearchReq(n.routingTable.getNeighbours(), requestID, budget, pattern, n.soc.GetAddress())
	if err != nil {
		return "", err
	}

	ticker := time.NewTicker(conf.Timeout)
	// wait for the first reply
	for {
		select {
		// return filename if channel is triggered and open
		case filename, isOpen := <-*searchChannel:
			if isOpen {
				return fmt.Sprintf("%v", filename), nil
			}
			return "", nil
		case <-ticker.C:
			// stops search after too many trials
			if trials >= conf.Retry {
				return "", nil
			}

			// expand budget
			budget *= conf.Factor

			// send search request to neighbours with updated budget
			err = n.sendSearchReq(n.routingTable.getNeighbours(), requestID, budget, pattern, n.soc.GetAddress())
			if err != nil {
				return "", err
			}
			trials++
		}
	}
}

// retrieves data from random peer with data available
func (n *node) getDataFromRemotePeer(metahash string, store storage.Store) ([]byte, error) {
	peers := n.catalog.getAddr(metahash)
	if peers == nil {
		return nil, xerrors.Errorf("metahash: %s not found", metahash)
	}

	// choose random peer with data
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(peers))
	randomPeer := peers[idx]
	myAddr := n.soc.GetAddress()
	requestID := xid.New().String()

	// compute address to send request
	peerAddr, inTable := n.routingTable.getAddr(randomPeer)
	if !inTable {
		return nil, xerrors.Errorf("address %v not found in routing table", randomPeer)
	}

	// create packet with data request
	dataRequestHeader := transport.NewHeader(myAddr, myAddr, randomPeer, 0)
	dataRequest := types.DataRequestMessage{
		RequestID: requestID,
		Key:       metahash,
	}

	pkt, err := n.createPkt(&dataRequestHeader, dataRequest)
	if err != nil {
		return nil, err
	}

	// create channel with requestID
	n.requestsHandler.createChannel(requestID)
	reqChannel, inTable := n.requestsHandler.getChannel(requestID)
	if !inTable {
		return nil, xerrors.Errorf("channel not found for requestID: %v", requestID)
	}

	// send request to chosen peer
	err = n.soc.Send(peerAddr, pkt, 0)
	if err != nil {
		return nil, err
	}

	// wait for timeout and check if a reply has been received or send request again
	timeout := n.conf.BackoffDataRequest.Initial
	ticker := time.NewTicker(timeout)
	repReceived := false
	trials := uint(1)
	for !repReceived {
		select {
		case <-*reqChannel:
			// stop loop in case of reply
			repReceived = true
		case <-ticker.C:
			// stop after too many trials
			if trials > n.conf.BackoffDataRequest.Retry {
				return nil, xerrors.Errorf("too many trials for requestID %v", requestID)
			}

			// send packet agin
			err = n.soc.Send(peerAddr, pkt, 0)
			if err != nil {
				return nil, err
			}

			// update timeout
			timeout *= time.Duration(n.conf.BackoffDataRequest.Factor)
			ticker = time.NewTicker(timeout)
			trials++
		}
	}

	// retrieve metafile
	metafile := store.Get(metahash)
	if metafile == nil {
		return nil, xerrors.Errorf("metahash : %v not found", metahash)
	}

	return metafile, nil
}

// sends requests to neighbours with given budget
func (n *node) sendSearchReq(neighbours []string, id string, budg uint, reg regexp.Regexp, org string) error {
	nbNeighbours := len(neighbours)
	myAddr := n.soc.GetAddress()
	if nbNeighbours <= 0 {
		return nil
	}

	distribution := make([]uint, nbNeighbours)

	// allocate same budget to each neighbour
	for budg > 0 {
		for i := 0; i < nbNeighbours; i++ {
			if budg > 0 {
				distribution[i]++
				budg--
			} else {
				break
			}
		}
	}

	// send search request to every neighbours with mapping budget
	for i, addr := range neighbours {
		if distribution[i] > 0 {

			searchRequestHeader := transport.NewHeader(myAddr, myAddr, addr, 0)
			searchRequest := types.SearchRequestMessage{
				RequestID: id,
				Origin:    org,
				Pattern:   reg.String(),
				Budget:    distribution[i],
			}
			err := n.createPktAndSend(&searchRequestHeader, searchRequest, addr)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// retrieves filenames locally and remotly
func (n *node) retrieveFilenames(requestID string, reg regexp.Regexp, filenames []string) []string {
	filenamesMap := make(map[string]void)

	// add local filenames to map
	store := n.conf.Storage.GetNamingStore()
	store.ForEach(func(key string, val []byte) bool {
		if reg.MatchString(key) {
			filenamesMap[key] = member
		}
		return true
	})

	remote := n.store.getFilenames(requestID)

	// add remote filenames to map
	for _, f := range remote {
		filenamesMap[f] = member
	}

	// compute filenames list to return
	for f := range filenamesMap {
		filenames = append(filenames, f)
	}

	return filenames
}

// returns filename if all chunks of data are available locally
func (n *node) localFullyKnowsFile(pattern regexp.Regexp) string {
	file := ""
	// iterate on naming store
	n.conf.Storage.GetNamingStore().ForEach(func(key string, val []byte) bool {
		if !pattern.MatchString(key) {
			return false
		}
		// check for if pair if entire file is available and return in case of matching
		store := n.conf.Storage.GetDataBlobStore()
		metafile := store.Get(string(val))
		if metafile != nil {
			chunkList := strings.Split(string(metafile), peer.MetafileSep)
			entireFileIsAvailable := true
			for _, chunkEncoded := range chunkList {
				if store.Get(chunkEncoded) == nil {
					entireFileIsAvailable = false
					break
				}
			}
			if entireFileIsAvailable {
				file = key
				return false
			}
		}
		return true
	})
	// return file if available or empty string in other case
	return file
}
