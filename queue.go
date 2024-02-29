package main

import (
	"sync"
)

const minCap = 64

type ThreadSafeQueue[T any] struct {
	// Think about type sizes, e.g. use int64 for capacity or int,
	// there is a slight probability that we ever go beyond max(int).
	front int
	back  int
	count int
	cap   int
	buf   []T
	mu    sync.Mutex
}

func NewQueue[T any](size ...int) *ThreadSafeQueue[T] {
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

	return &ThreadSafeQueue[T]{
		cap: cap,
		buf: buf,
	}
}

func (q *ThreadSafeQueue[T]) Cap() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.cap
}

func (q *ThreadSafeQueue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

func (q *ThreadSafeQueue[T]) Empty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count == 0
}

func (q *ThreadSafeQueue[T]) Left() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.cap - q.count
}

func (q *ThreadSafeQueue[T]) Push(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.grow()
	q.buf[q.back] = item
	q.back = q.nextIndex(q.back)
	q.count++
}

func (q *ThreadSafeQueue[T]) grow() {
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
			nCopied := copy(newBuf, q.buf[q.front:q.count])
			copy(newBuf[nCopied:], q.buf[:q.back])
		}

		q.cap = newCap
		q.buf = newBuf
		q.back = q.count
		q.front = 0
	}
}

func (q *ThreadSafeQueue[T]) TryPop(value *T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		return false
	}
	*value = q.Front()
	q.Pop()
	return true
}

// Not sure whether Pop should return an element or not.
// If we stick with std::queue implementation, it shouldn't.
// Normally Pop doesn't return anything.
func (q *ThreadSafeQueue[T]) Pop() T {
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

func (q *ThreadSafeQueue[T]) nextIndex(index int) int {
	if index == (q.cap - 1) {
		return 0
	}
	return index + 1
}

func (q *ThreadSafeQueue[T]) Front() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot retrieve Front element on empty queue.")
	}

	return q.buf[q.front]
}

func (q *ThreadSafeQueue[T]) Back() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot retrieve Back element on empty queue.")
	}

	// wrapped
	if q.back == 0 {
		return q.buf[q.count-1]
	}

	return q.buf[q.back-1]
}

// Allowed indices are in a range [0, q.count - 1]
func (q *ThreadSafeQueue[T]) Replace(index int, elem T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		panic("Cannot replace on an empty queue.")
	}

	if index >= q.count {
		panic("Cannot replace element at index [index]. Index out of range.")
	}

	pos := q.front
	for i := 0; i < index; i++ {
		pos = q.nextIndex(q.front)
	}

	// for debuggin
	if pos >= q.back {
		panic("Invalid position.")
	}

	q.buf[pos] = elem
}

// func (q *ThreadSafeQueue[T]) At(index int) T {
// 	q.mu.Lock()
// }

func (q *ThreadSafeQueue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == 0 {
		return
	}

	// No need to initialize all the elements to zero since we operate on indices (front/back) anyway
	// to push/pop elements.
	q.zeroMemebers()
}

func (q *ThreadSafeQueue[T]) Flush(res []T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	copy(res, q.buf)
	q.zeroMemebers()
}

func (q *ThreadSafeQueue[T]) zeroMemebers() {
	// Not sure whether we need to zero a capacity.
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
