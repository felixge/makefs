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
	closeErr    error
}

func newBroadcast() *broadcast {
	b := new(broadcast)
	b.cacheUpdate.L = b.cacheLock.RLocker()
	return b
}

func (b *broadcast) Write(buf []byte) (int, error) {
	b.cacheLock.Lock()
	defer b.cacheLock.Unlock()
	if b.closeErr == io.EOF {
		return 0, io.ErrClosedPipe
	} else if b.closeErr != nil {
		return 0, b.closeErr
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

		if b.closeErr != nil {
			return 0, b.closeErr
		}

		// aquires a new RLock() before returning
		b.cacheUpdate.Wait()
	}
	panic("unreachable")
}

func (b *broadcast) Close() error {
	return b.CloseWithError(nil)
}

func (b *broadcast) CloseWithError(err error) error {
	if err == nil {
		err = io.EOF
	}

	b.cacheLock.Lock()
	defer b.cacheLock.Unlock()

	b.closeErr = err
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
