package impl

import "sync"

// initialises store
func newStore() *store {
	return &store{
		storeMap: make(map[string]map[string]void),
	}
}

type store struct {
	sync.RWMutex
	storeMap map[string]map[string]void
}

// creates entry in storeMap with given requestID
func (s *store) createRequestMap(requestID string) {
	s.Lock()
	defer s.Unlock()

	s.storeMap[requestID] = make(map[string]void)
}

// maps filename to requestID if entry is created
func (s *store) addEntry(requestID string, filename string) {
	s.Lock()
	defer s.Unlock()

	_, inTable := s.storeMap[requestID]

	if inTable {
		s.storeMap[requestID][filename] = member
	}
}

// returns filenames from given requestID
func (s *store) getFilenames(requestID string) []string {
	s.Lock()
	defer s.Unlock()

	filenamesMap := s.storeMap[requestID]

	var filenames = make([]string, 0, len(filenamesMap))

	// retrieve list of filename from storeMap
	for f := range filenamesMap {
		filenames = append(filenames, f)
	}

	// delete entry corresponding to requestID in storeMap
	delete(s.storeMap, requestID)

	return filenames
}
