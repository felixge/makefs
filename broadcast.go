package makefs

import (
	"io"
	"sync"
)

// broadcast is a writer that broadcasts all writes to all registered clients.
// Additionally it caches all writes, so that new clients won't miss any
// data.
type broadcast struct {
	clientsLock  sync.RWMutex
	clients      []*client
	cacheLock    sync.RWMutex
	cache        []byte
	waiters int
	waitersLock sync.Mutex
	signal  chan interface{}
}

func newBroadcast() *broadcast {
	return &broadcast{signal: make(chan interface{})}
}

func (b *broadcast) Write(buf []byte) (int, error) {
	b.cacheLock.Lock()
	b.cache = append(b.cache, buf...)
	b.cacheLock.Unlock()

	// notify any goroutines wait()ing for a write
	b.broadcast()

	return len(buf), nil
}

func (b *broadcast) ReadAt(buf []byte, offset int64) (int, error) {
	for {
		b.cacheLock.RLock()
		if int(offset) < len(b.cache) {
			b.cacheLock.RUnlock()
			return copy(buf, b.cache[offset:]), nil
		}

		b.cacheLock.RUnlock()

		// wait for the next write
		b.wait()
	}
	panic("unreachable")
}

func (b *broadcast) broadcast() {
	b.waitersLock.Lock()
	defer b.waitersLock.Unlock()

	for i := 0; i < b.waiters; i++ {
		b.signal <- nil
	}
	b.waiters = 0
}

func (b *broadcast) wait() {
	b.waitersLock.Lock()
	b.waiters++
	b.waitersLock.Unlock()
	<-b.signal
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
