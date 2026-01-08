package buffer

import (
	"gcli/internal/config"
	"runtime"
	"sync/atomic"
)

type ringBuffer struct {
	wPos, rPos uint64
	closed     uint32
	buf        [64]config.Config
}

func NewRb() *ringBuffer {
	return &ringBuffer{
		wPos: 0,
		rPos: 0,
	}
}

func (b *ringBuffer) Write(c config.Config) {
	for {
		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		if w+1-r > 64 {
			runtime.Gosched()
			continue
		}

		b.buf[w%64] = c
		atomic.AddUint64(&b.wPos, 1)
		return
	}
}
func (b *ringBuffer) Read() config.Config {
	for {
		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		if r == w {
			if b.IsClosed() {
				return nil
			}
			runtime.Gosched()
			continue
		}

		val := b.buf[r%64]
		atomic.AddUint64(&b.rPos, 1)
		return val
	}
}

func (b *ringBuffer) Close() {
	atomic.StoreUint32(&b.closed, 1)
}
func (b *ringBuffer) IsClosed() bool {
	return atomic.LoadUint32(&b.closed) == 1
}
