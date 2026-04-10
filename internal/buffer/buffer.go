// Package buffer buffer.go contains ringbuffer implementation.
// Used spin-lock model.
// No mutex used.
package buffer

import (
	"runtime"
	"sync/atomic"
	"time"
)

// bufSize is a max size of buffer.
const bufSize uint64 = 10

// Buffer is a interface for ringbuffer.
type Buffer[T any] interface {
	// Write writes value to buffer.
	Write(val T)

	// Read reads value from buffer.
	Read() T

	// Close closes buffer.
	Close()

	// IsClosed returns true if buffer is closed.
	IsClosed() bool
}

// ringBuffer is a classic ringbuffer implementation.
type ringBuffer[T any] struct {
	// wPos and rPos are cursors for buffer.
	wPos, rPos uint64
	// closed is a flag for closed buffer.
	closed uint32
	// buf is a buffer for values.
	buf [bufSize]T
}

// nopBuffer is a buffer for no-op.
type nopBuffer[T any] struct {
	buf []T
}

// NewRb accepts type and returns new ringbuffer.
func NewRb[T any]() Buffer[T] {
	return &ringBuffer[T]{
		wPos: 0,
		rPos: 0,
	}
}

// NewNop accepts type and returns new nopbuffer.
func NewNop[T any]() Buffer[T] {
	return &nopBuffer[T]{
		buf: make([]T, 0),
	}
}

// spin is a helper function for spin-lock.
func spin(idx *int) {
	*idx++
	if *idx < 10 {
		runtime.Gosched()
	} else {
		time.Sleep(time.Millisecond)
		*idx = 0
	}
}

func (b *ringBuffer[T]) Write(val T) {
	idx := 0
	for {
		if b.IsClosed() {
			return
		}

		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		if w-r >= bufSize {
			spin(&idx)
			continue
		}

		b.buf[w%bufSize] = val
		atomic.AddUint64(&b.wPos, 1)
		return
	}
}

func (b *ringBuffer[T]) Read() T {
	idx := 0
	for {
		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		if r < w {
			val := b.buf[r%bufSize]
			atomic.AddUint64(&b.rPos, 1)
			return val
		}

		if b.IsClosed() {
			var zero T
			return zero
		}

		spin(&idx)
	}
}

func (b *ringBuffer[T]) Close() {
	atomic.StoreUint32(&b.closed, 1)
}

func (b *ringBuffer[T]) IsClosed() bool {
	return atomic.LoadUint32(&b.closed) == 1
}

func (b *nopBuffer[T]) Write(val T) {
}

func (b *nopBuffer[T]) Read() T {
	var zero T
	return zero
}

func (b *nopBuffer[T]) Close() {
}

func (b *nopBuffer[T]) IsClosed() bool {
	return false
}
