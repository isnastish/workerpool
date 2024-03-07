package main

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const displayMetrics = false

func TestThreadPool_TooHighWorkerCount(t *testing.T) {
	const threadCount = 256
	var expectedThreadCount = uint32(runtime.NumCPU())

	p := NewPool(displayMetrics)
	assert.Equal(t, p.maxThreads, expectedThreadCount)
}

func TestThreadPool_NegativeWorkerCount(t *testing.T) {
	const threadCount = 256
	var expectedThreadCount = uint32(runtime.NumCPU())
	p := NewPool(displayMetrics, threadCount)
	assert.Equal(t, p.maxThreads, expectedThreadCount)
}

func TestThreadPool_CorrectWorkerCount(t *testing.T) {
	const threadCount = 16
	p := NewPool(displayMetrics, threadCount)
	assert.EqualValues(t, p.maxThreads, threadCount)
}

func TestThreadPoo_NumberOfTasksEqualToWorkers(t *testing.T) {
	const threadCount = 8
	p := NewPool(displayMetrics, threadCount)

	var makeSureThatAllTasksExecuted atomic.Int32
	var sleepDuration = 2000 * time.Millisecond

	for index := 0; index < threadCount; index++ {
		p.SubmitTask(func() {
			makeSureThatAllTasksExecuted.Add(1)
			time.Sleep(sleepDuration)
		})
	}

	var start = time.Now()
	p.ProcessSubmittedTasks()
	var elapsed = time.Since(start)

	assert.EqualValues(t, makeSureThatAllTasksExecuted.Load(), threadCount)
	assert.EqualValues(t, p.metrics.tasksSubmitted, threadCount)
	assert.EqualValues(t, p.metrics.tasksQueued, 0)
	assert.Less(t, elapsed, 3000*time.Millisecond) // maybe 3000 is too strict, bump it to 4000 if the test starts to fail.
	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}

func TestThreadPoo_NumberOfTasksIsGreaterThenWorkerCount(t *testing.T) {
	const threadCount = 4
	const tasksCount = 16
	p := NewPool(displayMetrics, threadCount)

	var makeSureThatAllTasksExecuted atomic.Int32
	var sleepDuration = 2000 * time.Millisecond

	for index := 0; index < tasksCount; index++ {
		p.SubmitTask(func() {
			makeSureThatAllTasksExecuted.Add(1)
			time.Sleep(sleepDuration)
		})
	}

	var start = time.Now()
	p.ProcessSubmittedTasks()
	var elapsed = time.Since(start)

	assert.EqualValues(t, makeSureThatAllTasksExecuted.Load(), tasksCount)
	assert.EqualValues(t, p.metrics.tasksSubmitted, tasksCount)
	assert.EqualValues(t, p.metrics.tasksQueued, 12)
	assert.Less(t, elapsed, 9000*time.Millisecond)
	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}
