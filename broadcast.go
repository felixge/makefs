package makefs

import (
	"io"
	"sync"
)

// broadcast is a writer that broadcasts all writes to all registered clients.
// Additionally it caches all writes, so that new clients won't miss any
// data.
type broadcast struct {
	cacheLock   sync.RWMutex
	cacheUpdate sync.Cond
	cache       []byte
	closed      bool
}

func newBroadcast() *broadcast {
	b := new(broadcast)
	b.cacheUpdate.L = b.cacheLock.RLocker()
	return b
}

func (b *broadcast) Write(buf []byte) (int, error) {
	b.cacheLock.Lock()
	defer b.cacheLock.Unlock()
	if (b.closed) {
		return 0, io.ErrClosedPipe
	}

	b.cache = append(b.cache, buf...)
	b.cacheUpdate.Broadcast()

	return len(buf), nil
}

func (b *broadcast) ReadAt(buf []byte, offset int64) (int, error) {
	b.cacheLock.RLock()
	defer b.cacheLock.RUnlock()

	for {
		if int(offset) < len(b.cache) {
			return copy(buf, b.cache[offset:]), nil
		}

		if b.closed {
			return 0, io.EOF
		}

		// aquires a new RLock() before returning
		b.cacheUpdate.Wait()
	}
	panic("unreachable")
}

func (b *broadcast) Close() error {
	b.cacheLock.Lock();
	defer b.cacheLock.Unlock();

	b.closed = true
	b.cacheUpdate.Broadcast()

	return nil
}

func (b *broadcast) Client() *client {
	return &client{broadcast: b}
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
