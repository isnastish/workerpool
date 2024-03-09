package main

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

const displayMetrics = false

type integer interface {
	int | int16 | int32 | int64
}

type integerOrString interface {
	integer | string
}

func sliceHasValue[T integerOrString](s []T, v T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == v {
			return true
		}
	}
	return false
}

// Distribute chunks between multiple tasks, submitt them for processing by thread pool
func distributeWorkByChunks[T integer](data []T, p *ThreadPool, resultsCh chan int64, chunkSize int) {
	dataSize := len(data)
	nChunks := dataSize / chunkSize
	computeSum := func(start, end int) int64 {
		var res int64
		for i := start; i < end; i++ {
			res += int64(data[i])
		}
		return res
	}

	for i := 0; i < nChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize

		p.SubmitTask(func() {
			resultsCh <- computeSum(start, end)
		})
	}

	// Compute sum for the remaining chunk.
	if dataSize%chunkSize != 0 {
		start := nChunks * chunkSize
		end := dataSize

		p.SubmitTask(func() {
			resultsCh <- computeSum(start, end)
		})
	}
}

// Populate slice with of length len(s) with elements returned by callable
func populate[T integer](s []T, callable func(int) T) T {
	var sum T
	for i := 0; i < len(s); i++ {
		v := callable(i)
		s[i] = v
		sum += v
	}
	return sum
}

func TestClipThreadCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	const maxThreads = 256
	var expectedThreadCount = uint32(runtime.NumCPU())
	p := NewPool(displayMetrics, maxThreads)
	assert.Equal(t, p.maxThreads, expectedThreadCount)
}

func TestCorrectWorkerCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	const maxThreads = 16
	p := NewPool(displayMetrics, maxThreads)
	assert.EqualValues(t, p.maxThreads, maxThreads)
}

func TestExample(t *testing.T) {
	defer goleak.VerifyNone(t)

	const maxThreads uint32 = 8

	data := []int{ // fibonacci numbers
		0, 1, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89,
		144, 233, 377, 610, 987, 1597, 2584, 4181,
		6765, 10946, 17711, 28657, 46368, 75025,
		121393, 196418, 317811, 514229,
	}
	dataSize := uint32(len(data))

	p := NewPool(displayMetrics, maxThreads)
	recvData := make([]int, 0, dataSize)
	resCh := make(chan int, dataSize)

	for i := 0; i < int(dataSize); i++ {
		index := i
		p.SubmitTask(func() {
			resCh <- data[index]
		})
	}

	p.ProcessSubmittedTasks()

	close(resCh)

	for v := range resCh {
		recvData = append(recvData, v)
		assert.True(t, sliceHasValue(data, v))
	}

	assert.ElementsMatch(t, data, recvData)

	m := p.GetMetrics()
	assert.Equal(t, m.tasksSubmitted, dataSize)
	assert.Equal(t, m.tasksDone, dataSize)
	assert.Equal(t, m.threadsFinished, m.threadsSpawned)

	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}

func TestExample2(t *testing.T) {
	defer goleak.VerifyNone(t)

	const maxThreads uint32 = 4

	data := []string{
		"Red", "lazy", "fox", "jumped", "over", "the", "long", "wooden", "fance", "!",
		"Green", "fatty", "frog", "was", "sitting", "near", "the", "old", "lake", ".",
	}

	dataSize := uint32(len(data))
	p := NewPool(displayMetrics, maxThreads)

	recvData := make([]string, 0, dataSize)
	resCh := make(chan string, dataSize)

	for i := 0; i < int(dataSize); i++ {
		index := i
		p.SubmitTask(func() {
			resCh <- data[index]
		})
	}

	p.ProcessSubmittedTasks()

	close(resCh)

	for str := range resCh {
		recvData = append(recvData, str)
		assert.True(t, sliceHasValue(data, str))
	}

	assert.ElementsMatch(t, data, recvData)

	m := p.GetMetrics()
	assert.Equal(t, m.tasksSubmitted, dataSize)
	assert.Equal(t, m.tasksDone, dataSize)
	assert.Equal(t, m.threadsFinished, m.threadsSpawned)

	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}

// T16xC16 - 16 threads involved to compute sum of 16 chunks
func BenchmarkConcurrentAccumulate_T16xC16(b *testing.B) {
	// goleak doesn't work correctly with benchmarks.
	// This is the workaround to avoid goleak panicing on goroutines on top of the stack.
	// https://github.com/uber-go/goleak/issues/77
	defer goleak.VerifyNone(b,
		goleak.IgnoreTopFunction("testing.(*B).run1"),
		goleak.IgnoreTopFunction("testing.(*B).doBench"),
	)

	b.ResetTimer()

	const maxThreads = 16
	const chunkSize = 256
	const dataSize = 4096

	var totalSum int64

	data := make([]int64, dataSize)
	_ = populate(data, func(i int) int64 { return int64((i + 1) << 1) })
	p := NewPool(displayMetrics, maxThreads)
	nChunks := (dataSize / chunkSize)

	if dataSize%chunkSize != 0 {
		nChunks += 1
	}

	for i := 0; i < b.N; i++ {
		resCh := make(chan int64, nChunks)

		distributeWorkByChunks(data, p, resCh, chunkSize)
		p.ProcessSubmittedTasks()

		close(resCh)

		for chunkSum := range resCh {
			totalSum += chunkSum
		}
	}
}
