package main

import (
	"fmt"
	"sync"
)

// Minimum default capacity.
const minCap = 64

type Queue[T any] struct {
	front int
	back  int
	count int
	cap   int
	buf   []T
	mu    sync.Mutex
}

func NewQueue[T any](size ...int) *Queue[T] {
	var cap int
	var buf []T
	if len(size) > 0 {
		if isPowerOf2(uint32(size[0])) {
			cap = size[0]
		} else {
			cap = int(ceilPow2(uint32(size[0])))
		}
		buf = make([]T, cap)
	}

	return &Queue[T]{
		cap: cap,
		buf: buf,
	}
}

func (q *Queue[T]) Cap() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.cap
}

func (q *Queue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

func (q *Queue[T]) Empty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count == 0
}

func (q *Queue[T]) Left() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.cap - q.count
}

func (q *Queue[T]) Push(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.grow()
	q.buf[q.back] = item
	q.back = q.nextIndex(q.back)
	q.count++
}

func (q *Queue[T]) TryPop(value *T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		return false
	}
	*value = q.buf[q.front]

	var zeroValue T
	q.buf[q.front] = zeroValue
	q.front = q.nextIndex(q.front)
	q.count--

	return true
}

func (q *Queue[T]) Pop() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot Pop on empty queue.")
	}

	var zeroValue T

	res := q.buf[q.front]
	q.buf[q.front] = zeroValue
	q.front = q.nextIndex(q.front)
	q.count--

	return res
}

func (q *Queue[T]) Front() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot retrieve Front element on empty queue.")
	}

	return q.buf[q.front]
}

func (q *Queue[T]) Back() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot retrieve Back element on empty queue.")
	}

	if q.back == 0 {
		return q.buf[q.count-1]
	}

	return q.buf[q.back-1]
}

func (q *Queue[T]) Replace(index int, elem T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot replace, queue is empty.")
	}

	if index >= q.count {
		panic(fmt.Sprintf("Cannot replace element at index [%d]. Index out of range.", index))
	}

	pos := q.front
	for i := 0; i < index; i++ {
		pos = q.nextIndex(pos)
	}

	q.buf[pos] = elem
}

func (q *Queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		return
	}

	q.zeroMemebers()
}

func (q *Queue[T]) Flush(res []T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 || res == nil {
		return
	}

	if q.back > q.front {
		copy(res, q.buf[q.front:q.back])
	} else {
		nCopied := copy(res, q.buf[q.front:q.cap])
		copy(res[nCopied:], q.buf[0:q.back])
	}

	q.zeroMemebers()
}

func (q *Queue[T]) grow() {
	if q.cap == 0 {
		q.cap = minCap
		q.buf = make([]T, q.cap)
	}

	if q.count >= q.cap {
		newCap := q.cap << 1
		newBuf := make([]T, newCap)

		if q.back > q.front {
			copy(newBuf, q.buf[q.front:q.back])
		} else {
			nCopied := copy(newBuf, q.buf[q.front:q.cap])
			copy(newBuf[nCopied:], q.buf[:q.back])
		}

		q.cap = newCap
		q.buf = newBuf
		q.back = q.count
		q.front = 0
	}
}

func (q *Queue[T]) nextIndex(index int) int {
	if index == (q.cap - 1) {
		return 0
	}
	return index + 1
}

func (q *Queue[T]) zeroMemebers() {
	// Think about how we can avoid doing memory allocation.
	zeroBuf := make([]T, q.count)
	copy(q.buf, zeroBuf)

	q.front = 0
	q.back = 0
	q.count = 0
}

// Round up to the next power of 2
func ceilPow2(x uint32) uint32 {
	x = x - 1

	x = x | (x >> 1)
	x = x | (x >> 2)
	x = x | (x >> 4)
	x = x | (x >> 8)
	x = x | (x >> 16)

	return x + 1
}

// Check whether uint32 is a power of 2.
func isPowerOf2(x uint32) bool {
	if x == 0 {
		return false
	}
	return x&(x-1) == 0
}
