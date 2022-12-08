package impl

import "sync"

// initialises channels handler
func newChannelsHandler() *channelsHandler {
	return &channelsHandler{
		channels: make(map[string]*chan interface{}),
	}
}

type channelsHandler struct {
	sync.Mutex
	channels map[string]*chan interface{}
}

// creates a channel for specific ID
func (cH *channelsHandler) createChannel(id string) {
	cH.Lock()
	defer cH.Unlock()

	// create channel and add it to map to keep track
	channel := make(chan interface{})
	cH.channels[id] = &channel
}

// returns channel corresponding to given ID
func (cH *channelsHandler) getChannel(id string) (*chan interface{}, bool) {
	cH.Lock()
	defer cH.Unlock()

	// retrieve channel from map if it exists
	channel, inTable := cH.channels[id]
	return channel, inTable
}

// closes all channels in handler
func (cH *channelsHandler) close() {
	cH.Lock()
	defer cH.Unlock()

	// iterate on map, close each channel and delete it from map
	for id, channel := range cH.channels {
		close(*channel)
		delete(cH.channels, id)
	}
}

// deletes channel for specific IDs
func (cH *channelsHandler) deleteChannel(id string) {
	cH.Lock()
	defer cH.Unlock()

	// retrieve channel if it exists
	channel, inTable := cH.channels[id]
	if !inTable {
		return
	}

	// close corresponding channel and delete from map
	close(*channel)
	delete(cH.channels, id)
}

// triggers channel and deletes it
func (cH *channelsHandler) triggerChannel(id string, trigger interface{}) {
	cH.Lock()
	defer cH.Unlock()

	channel, inTable := cH.channels[id]

	// trigger channel
	if inTable {
		*channel <- trigger
	}

	// delete from map
	delete(cH.channels, id)
}
