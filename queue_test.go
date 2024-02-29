package main

import (
	"fmt"
	"strconv"
	_ "sync"
	_ "sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper struct for testing aggregate types.
type Aggregate struct {
	i32 int
	str string
}

// Helper function, pushes n elements into a queue.
// Returns a slice of pushed elements.
func pushN[T any](q *ThreadSafeQueue[T], n int, f func(int) T) []T {
	res := make([]T, n)
	for i := 0; i < n; i++ {
		res[i] = f(i)
		q.Push(res[i])
	}
	return res
}

// Helper function. Pops (n) elements from the queue
// and returns a slice of them.
func popN[T any](q *ThreadSafeQueue[T], n int) []T {
	res := make([]T, n)
	for i := 0; i < n; i++ {
		res[i] = q.Pop()
	}
	return res
}

func TestQueue_CreationNoCapacity(t *testing.T) {
	q := NewQueue[string]()

	assert.EqualValues(t, q.Size(), 0)
	assert.EqualValues(t, q.Cap(), 0)
}

func TestQueue_CreationUseDefaultCapacity(t *testing.T) {
	q := NewQueue[string]()

	q.Push("baz")

	assert.EqualValues(t, q.Cap(), minCap)
	assert.EqualValues(t, q.Size(), 1)
	assert.EqualValues(t, q.front, 0)
	assert.EqualValues(t, q.back, 1)
}

func TestQueue_CreationCustomCapacity(t *testing.T) {
	const cap = 777
	var expectedCap = ceilPow2(cap) // round Up to the next power of 2.

	q := NewQueue[string](cap)

	assert.EqualValues(t, q.Cap(), expectedCap)
	assert.EqualValues(t, q.Size(), 0)
}

func TestQueue_PushN(t *testing.T) {
	const N = 1 << 10
	{
		q := NewQueue[int](N)

		res := pushN(q, N, func(i int) int { return i*10 + (i << 1) })

		assert.EqualValues(t, q.Cap(), N)
		assert.EqualValues(t, q.Size(), N)
		assert.EqualValues(t, q.front, q.back)

		assert.ElementsMatch(t, q.buf, res)
	}

	{
		q := NewQueue[string](N)

		res := pushN(q, N/2, func(i int) string { return "push_N:" + strconv.Itoa(i) })

		assert.EqualValues(t, q.Cap(), N)
		assert.EqualValues(t, q.Size(), N/2)
		assert.EqualValues(t, q.front, 0)
		assert.EqualValues(t, q.back, N/2)

		assert.ElementsMatch(t, q.buf[:N/2], res)
	}
}

func TestQueue_ForceToGrow(t *testing.T) {
	const N = 16
	q := NewQueue[int](N)

	res := pushN(q, N, func(i int) int { return i * 10 })
	res = append(res, pushN(q, N/2, func(i int) int { return (i + 10) * 10 })...)

	assert.EqualValues(t, q.Cap(), N<<1)
	assert.EqualValues(t, q.Size(), N+N/2)
	assert.EqualValues(t, q.front, 0)
	assert.EqualValues(t, q.back, q.Size())

	// ignore capacity
	assert.ElementsMatch(t, q.buf[:q.Size()], res)
}

func TestQueue_PushPop(t *testing.T) {
	const N = 8
	q := NewQueue[string](N)

	pushN(q, N, func(i int) string { return "push_N:" + strconv.Itoa(i) })

	assert.Equal(t, q.Front(), "push_N:0")
	assert.Equal(t, q.Back(), "push_N:7")

	assert.Equal(t, q.Pop(), "push_N:0")
	assert.Equal(t, q.Pop(), "push_N:1")
	assert.Equal(t, q.Pop(), "push_N:2")
	assert.Equal(t, q.Pop(), "push_N:3")
	assert.Equal(t, q.Pop(), "push_N:4")
	assert.Equal(t, q.Pop(), "push_N:5")

	assert.Equal(t, q.Front(), "push_N:6")
}

func TestQueue_PopN(t *testing.T) {
	const N = 16
	q := NewQueue[string](N)

	// Insert N-1 elements so q.back doesn't wrap to 0.
	pushRes := pushN(q, N-1, func(i int) string { return "push_N:" + strconv.Itoa(i*10) })
	popN(q, N/2)

	assert.EqualValues(t, q.front, N/2)
	assert.ElementsMatch(t, q.buf[q.front:q.back], pushRes[q.front:])
}

func TestQueue_WrapBackIndex(t *testing.T) {
	const N = 16

	q := NewQueue[int](N)

	// After pushing:
	// [1, 32768], N = 16
	//  |
	// (front/back)
	pushRes := pushN(q, N, func(i int) int { return 1 << i })

	assert.EqualValues(t, q.back, 0)

	// After pop:
	// [0, 0, 0, 0, 16, 32 ... 32768]
	//  |            |
	// back        front
	popRes := popN(q, N/4)

	assert.ElementsMatch(t, pushRes[:N/4], popRes)

	assert.EqualValues(t, q.front, N/4)
	assert.EqualValues(t, q.Left(), N/4)

	// After pushing two more elements.
	// [2, 4, 0, 0, 16, 32 ... 32768]
	//  |            |
	// back        front
	pushN(q, N/8, func(i int) int { return 2 << i })

	assert.EqualValues(t, q.back, N/8)
	assert.EqualValues(t, q.Left(), N/8)
}

func TestQueue_WrapFrontIndex(t *testing.T) {
	const N = 8
	q := NewQueue[int](N)

	// After push:
	//  [2 4 8 16 32 64 128 256]
	//   |
	// (front/back)
	pushN(q, N, func(i int) int { return 2 << i })

	// After pop:
	// [0, 0, 0, 0, 0, 0, 128, 256]
	//  |                  |
	// back               front
	popN(q, N-2)

	assert.EqualValues(t, q.front, N-2)
	assert.EqualValues(t, q.back, 0)
	assert.Equal(t, q.Front(), 128)

	// After push:
	// [4 8 16 32, 0, 0, 128, 256]
	//             |      |
	//            back   front
	pushN(q, N/2, func(i int) int { return 2 << (i + 1) })

	assert.EqualValues(t, q.Back(), 32)
	assert.EqualValues(t, q.Left(), 2)

	// After pop the front index will wrap:
	// [0, 0, 0, 32, 0, 0, 0, 0]
	//           |    \
	//         front  back
	popN(q, N-3)

	assert.EqualValues(t, q.front, N-5)
	assert.EqualValues(t, q.Size(), 1)
	assert.EqualValues(t, q.Front(), 32)
	assert.EqualValues(t, q.Back(), q.Front())
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue[Aggregate]()
	assert.True(t, q.Empty())
}

func TestQueue_MakeEmpty(t *testing.T) {
	const N = 4
	q := NewQueue[Aggregate]()

	pushN(q, N, func(i int) Aggregate {
		return Aggregate{i32: (1 << i), str: "push_N:" + strconv.Itoa(i)}
	})

	popN(q, N-1)
	assert.EqualValues(t, q.Size(), 1)

	popN(q, 1)
	assert.True(t, q.Empty())
}

func TestQueue_CopyTwoChunksWhenQueueIsFull(t *testing.T) {
	const N = 4
	q := NewQueue[int](N)

	q.Push(-124)
	q.Push(99)
	q.Push(178)
	q.Push(44)

	q.Pop()

	/* Queue should have the following structure:
	[0,   99, 178, 44]
	 |    |
	back front

	The next time we push an element, it will be inserted at index 0.
	*/

	q.Push(33)

	expected := []int{33, 99, 178, 44}
	assert.ElementsMatch(t, q.buf, expected)

	oldCount := q.Size()
	q.Push(77)

	/*
		Doubles the capacity, allocates a new buffer and copies two chunks into that buffer.
		firstChunk := q.buf[q.front:q.count]
		secondChunk := q.buf[:q.back]

		n := copy(newBuf, firstChunk)
		copy(newBuf[n:], secondChunk)

		[99, 178, 44, 33, 77, 0, 0, 0]
		 |                	  |
		 front               back
	*/

	expected = []int{99, 178, 44, 33, 77, 0, 0, 0}
	assert.ElementsMatch(t, q.buf, expected)

	assert.EqualValues(t, q.front, 0)
	assert.EqualValues(t, q.back, oldCount+1)
}

func TestQueue_SingleTryPop(t *testing.T) {
	const N = 16
	q := NewQueue[string](N)

	res := pushN(q, N/4, func(i int) string { return "push_N:" + strconv.Itoa(i) })
	assert.ElementsMatch(t, res, q.buf[0:N/4])

	nextFront := q.nextIndex(q.front)
	oldCount := q.count

	var v string
	assert.Equal(t, q.TryPop(&v), true)
	assert.Equal(t, v, "push_N:0")
	assert.Equal(t, q.count, oldCount-1)
	assert.Equal(t, q.front, nextFront)
}

func TestQueue_MultipleTryPop(t *testing.T) {
	const N = 4
	q := NewQueue[int](N)

	pushN(q, N, func(i int) int { return i << 1 })

	for i := 0; i < N; i++ {
		var v int

		nextFront := q.nextIndex(q.front)
		oldCount := q.count

		assert.Equal(t, q.TryPop(&v), true)
		assert.Equal(t, v, i<<1)
		assert.Equal(t, q.count, oldCount-1)
		assert.Equal(t, nextFront, q.front)
	}

	var v int
	assert.Equal(t, q.TryPop(&v), false)
}

// func TestQueue_ThreadSafety(t *testing.T) {
// 	const N = 32
// 	const halfN = N / 2
// 	const quarterN = N / 4

// 	var q = NewQueue[int](N)
// 	var wg = sync.WaitGroup{}

// 	var nPushed atomic.Int32
// 	var nPopped atomic.Int32

// 	wg.Add(1)
// 	go func() {
// 		pushN(q, N, func(i int) int { return i << 1 })
// 		nPushed.Add(N)
// 		wg.Done()
// 	}()

// 	wg.Add(1)
// 	go func() {
// 		popN(q, halfN)
// 		nPopped.Add(halfN)
// 		wg.Done()
// 	}()

// 	wg.Add(1)
// 	go func() {
// 		pushN(q, quarterN, func(i int) int { return (i*10 + 1) << 1 })
// 		nPushed.Add(quarterN)
// 		wg.Done()
// 	}()

// 	wg.Add(1)
// 	go func() {
// 		popN(q, halfN)
// 		nPopped.Add(halfN)
// 		wg.Done()
// 	}()

// 	wg.Wait()
// 	assert.Equal(t, (nPushed.Load() - nPopped.Load()), int32(q.count))
// }

func TestQueue_FlushNoWrapping(t *testing.T) {
	const N = 8
	const halfN = N / 2

	q := NewQueue[int]()
	res := pushN(q, halfN, func(i int) int { return i << 1 })
	expectedBuf := make([]int, halfN)
	copy(expectedBuf, res)

	assert.Equal(t, q.count, halfN)

	flushRes := make([]int, halfN)
	q.Flush(flushRes)

	assert.ElementsMatch(t, expectedBuf, flushRes)

	assert.Equal(t, q.count, 0)
	assert.Equal(t, q.front, 0)
	assert.Equal(t, q.back, 0)
}

func TestQueue_FlushWithWrapping(t *testing.T) {
	const N = 8
	const halfN = N / 2

	q := NewQueue[string]()
	res0 := pushN(q, N, func(i int) string { return "pushN:" + strconv.Itoa(i<<1) })
	popN(q, halfN)
	res1 := pushN(q, halfN-1, func(i int) string { return "pushN:" + strconv.Itoa((i*10)<<1) })
	const expectedSize = N - halfN + halfN - 1
	expectedBuf := make([]string, expectedSize)
	n := copy(expectedBuf, res0[halfN:])
	copy(expectedBuf[n:], res1)

	assert.Equal(t, q.count, expectedSize)

	flushRes := make([]string, expectedSize)
	q.Flush(flushRes)

	assert.ElementsMatch(t, expectedBuf, flushRes)

	assert.Equal(t, q.count, 0)
	assert.Equal(t, q.front, 0)
	assert.Equal(t, q.back, 0)
}

func TestQueue_Clear(t *testing.T) {
	const halfCap = minCap / 2
	q := NewQueue[int]()

	res := pushN(q, halfCap, func(i int) int { return i << 1 })
	assert.ElementsMatch(t, q.buf[0:halfCap], res)

	q.Clear()

	assert.Equal(t, q.count, 0)
	assert.Equal(t, q.front, 0)
	assert.Equal(t, q.back, 0)

	assert.ElementsMatch(t, q.buf, make([]int, minCap))
}

func TestQueue_ReplaceOnEmptyQueueShouldPanic(t *testing.T) {
	const N = 4
	q := NewQueue[string](N)

	defer func() {
		r := recover()
		assert.True(t, r != nil)
	}()

	q.Replace(0, "NewString")
}

func TestQueue_ReplaceIndexOutOfRange(t *testing.T) {
	const N = 4
	q := NewQueue[string](N)

	defer func() {
		r := recover()
		assert.True(t, r != nil)
	}()

	q.Push("push_n:0")
	q.Push("push_n:1")

	assert.Equal(t, q.count, 2)

	q.Replace(3, "push_n:9999")
}

func TestQueue_ReplaceNoWrapping(t *testing.T) {
	const N = 4
	q := NewQueue[int](N)

	res := pushN(q, N, func(i int) int { return i << 1 })
	assert.ElementsMatch(t, q.buf, res)

	q.Replace(0, 10<<1)
	assert.Equal(t, q.buf[q.front], 10<<1)

	q.Replace(N-1, 12<<1)
	assert.Equal(t, q.Back(), 12<<1)
}

func TestQueue_ReplaceWithWrapping(t *testing.T) {
	const N = 8
	const halfN = N / 2

	q := NewQueue[int](N)

	res0 := pushN(q, N, func(i int) int { return i << 1 })
	popN(q, halfN)
	res1 := pushN(q, halfN-1, func(i int) int { return ((i * 10) + 1) << 1 })

	const size = N - halfN + halfN - 1
	expectedBuf := make([]int, size)
	n := copy(expectedBuf, res0[halfN:])
	copy(expectedBuf[n:], res1)

	q.Replace(0, 10<<1)
	assert.Equal(t, 10<<1, q.buf[q.front])

	fmt.Println(q.buf)

	q.Replace(q.count-1, 15<<1)
	assert.Equal(t, 15<<1, q.Back())
}
