package impl

import (
	"sync"

	"go.dedis.ch/cs438/peer"
)

// initialises catalog
func newCatalog() *catalog {
	return &catalog{
		catalogMap: make(peer.Catalog),
	}
}

type catalog struct {
	sync.RWMutex
	catalogMap peer.Catalog
}

// adds new entry in catalog with given key/address
func (c *catalog) addEntry(key string, addr string) {
	c.Lock()
	defer c.Unlock()

	_, inMap := c.catalogMap[key]
	if !inMap {
		c.catalogMap[key] = make(map[string]struct{})
	}

	c.catalogMap[key][addr] = struct{}{}
}

// returns addresses from key in catalog
func (c *catalog) getAddr(key string) []string {
	c.RLock()
	defer c.RUnlock()

	catalogMap, inMap := c.catalogMap[key]

	// return a copy of addresses available in catalog given the key
	if inMap {
		copyList := make([]string, len(catalogMap))

		i := 0
		for addr := range catalogMap {
			copyList[i] = addr
			i++
		}

		return copyList
	}

	return nil
}

// returns the catalog
func (c *catalog) getCatalog() peer.Catalog {
	c.RLock()
	defer c.RUnlock()

	copyCatalog := make(peer.Catalog)

	// make copy of catalog
	for key, addr := range c.catalogMap {
		copyCatalog[key] = addr
	}

	return copyCatalog
}
