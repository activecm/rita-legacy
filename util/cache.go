package util

import "sync"

type (

	// Cache is a simple cache mechanism to avoid db lookups
	Cache struct {
		lock  *sync.Mutex
		cache map[string]bool
	}
)

// BSONFAIL is a bson.ObjectId failure indicator
const (
	BSONFAIL = "FAIL"
)

// NewCache creates a new connection cache
func NewCache() Cache {
	c := make(map[string]bool)
	return Cache{
		lock:  new(sync.Mutex),
		cache: c,
	}
}

// Lookup a value in the cache, return true if present or false if not found.
// Once a lookup has been performed the value is cached so that the next lookup
// will cause the cache to return true.
func (c Cache) Lookup(hash string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.cache[hash]
	if !ok {
		c.cache[hash] = true
	}
	return ok
}

func (c Cache) Keys() []string {
	c.lock.Lock()
	defer c.lock.Unlock()
	var out []string
	for v := range c.cache {
		out = append(out, v)
	}
	return out
}
