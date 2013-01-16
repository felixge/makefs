package makefs

import (
	"io"
	"sync"
)

// broadcast is a writer that broadcasts all writes to all registered clients.
// Additionally it caches all writes, so that new clients won't miss any
// data.
type broadcast struct {
	clientsLock sync.RWMutex
	clients     []*client
	cacheLock   *sync.RWMutex
	cacheCond   *sync.Cond
	cache       []byte
}

func newBroadcast() *broadcast {
	cacheLock := &sync.RWMutex{}
	return &broadcast{
		cacheLock: cacheLock,
		cacheCond: sync.NewCond(cacheLock.RLocker()),
	}
}

func (b *broadcast) Write(buf []byte) (int, error) {
	b.cacheLock.Lock()
	b.cache = append(b.cache, buf...)
	b.cacheLock.Unlock()

	b.cacheCond.Broadcast()

	return len(buf), nil
}

func (b *broadcast) ReadAt(buf []byte, offset int64) (int, error) {
	b.cacheLock.RLock()
	defer b.cacheLock.RUnlock()

	for {
		if int(offset) < len(b.cache) {
			return copy(buf, b.cache[offset:]), nil
		}

		// aquires a new RLock() before returning
		b.cacheCond.Wait()
	}
	panic("unreachable")
}

func (b *broadcast) Client() io.ReadCloser {
	client := &client{broadcast: b}
	b.clientsLock.Lock()
	b.clients = append(b.clients, client)
	b.clientsLock.Unlock()
	return client
}

type client struct {
	offset    int64
	broadcast *broadcast
}

func (c *client) Read(buf []byte) (int, error) {
	n, err := c.broadcast.ReadAt(buf, c.offset)
	c.offset += int64(n)
	return n, err
}

func (c *client) Close() error {
	return nil
}
