package main

// TODO: Unify error handling in functions like At/Replace when a queue is either empty or pos is out of range.

import "fmt"

const queueMinCap = 64

type Queue[T any] struct {
	front int32
	back  int32
	count int32
	cap   uint32
	buf   []T
}

func NewQueue[T any](size ...uint32) *Queue[T] {
	// If capacity is specified, round it up to the next power of 2
	// and make a buf which is that big.
	// Otherwise do the allocation when we actually start pushing elements.
	var cap uint32
	var buf []T
	if len(size) > 0 {
		if isPowerOf2(size[0]) {
			cap = size[0]
		} else {
			cap = ceilPow2(size[0])
		}
		buf = make([]T, cap)
	}

	return &Queue[T]{
		cap: cap,
		buf: buf,
	}
}

func (q *Queue[T]) Cap() uint32 {
	if q == nil {
		return 0
	}
	return q.cap
}

func (q *Queue[T]) Empty() bool {
	if q == nil {
		return false
	}
	return q.count == 0
}

func (q *Queue[T]) Len() int32 {
	if q == nil {
		return 0
	}
	return q.count
}

// How many open slots left until the capacity will be doubled.
func (q *Queue[T]) Left() int32 {
	if q == nil {
		return 0
	}
	return int32(q.Cap()) - q.Len()
}

func (q *Queue[T]) Push(item T) {
	if q == nil {
		return
	}

	q.grow()

	q.buf[q.back] = item
	q.back = q.advance(q.back)
	q.count++
}

func (q *Queue[T]) grow() {
	if q.cap == 0 {
		q.cap = queueMinCap
		q.buf = make([]T, q.cap)
	}

	if q.count >= int32(q.cap) {
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

// func (q *Queue[T]) Resize(newCap int32) {
// 	// NOTE: Should use grow internally.
// }

func (q *Queue[T]) Front() T {
	if q.count <= 0 {
		panic("Front() is called on empty queue.")
	}

	return q.buf[q.front]
}

func (q *Queue[T]) Back() T {
	if q.count <= 0 {
		panic("Back() is called on empty queue.")
	}

	if q.back == 0 {
		return q.buf[q.count-1]
	}

	return q.buf[q.back-1]
}

// func (q *Queue[T]) At(pos int) T {
// 	if q.Empty() {
// 		panic(fmt.Sprintf("At(%d) is called on empty queue.", pos))
// 	}

// 	if pos < int(q.front) || pos >= int(q.count) {
// 		panic(fmt.Sprintf("Index: [%d] is out of range.", pos))
// 	}

// 	return q.buf[pos]
// }

func (q *Queue[T]) Replace(pos int, item T) {
	if q.Empty() {
		panic(fmt.Sprintf("Replace(%d) is called on empty queue.", pos))
	}

	if pos < int(q.front) || pos >= int(q.count) {
		panic(fmt.Sprintf("Index: [%d] is out of range.", pos))
	}

	q.buf[pos] = item
}

func (q *Queue[T]) Pop() T {
	if q.Empty() {
		panic("Tried to PopFront() on an empty queue.")
	}

	// NOTE: Since the value will be overwritten anyhow,
	// there is no point to zero initialize it.
	var zeroValue T

	res := q.buf[q.front]
	q.buf[q.front] = zeroValue
	q.front = q.advance(q.front)
	q.count--

	return res
}

func (q *Queue[T]) advance(pos int32) int32 {
	if pos == int32(q.cap)-1 {
		return 0
	}
	return pos + 1
}

func (q *Queue[T]) Clear() {
	if q.Empty() {
		return
	}

	// NOTE: Not sure whether we need to zero-initialize all the elements
	// since we realy on indices anyway (q.front and q.back) in order to operate on elements.
	q.front = 0
	q.back = 0
	q.count = 0
}

// func (q *Queue[T]) PopBack() T {
// 	// NOTE: Removing elements from the back doesn't have the same problem as removing them from the back.
// 	// Because that place will be occupied with a new element the next time we call Push()
// 	if q.Empty() {
// 		panic("Tried to PopBack() on an empty queue.")
// 	}

// 	oldBack := q.back
// 	q.back--
// 	q.count--

// 	return q.buf[oldBack]
// }

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
