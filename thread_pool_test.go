package main

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThreadPool_TooHighWorkerCount(t *testing.T) {
	const THREADS_COUNT = 256
	var expectedThreadCount = int32(runtime.NumCPU())

	p := NewPool(THREADS_COUNT)
	assert.Equal(t, p.maxThreads, expectedThreadCount)
}

func TestThreadPool_NegativeWorkerCount(t *testing.T) {
	var expectedThreadCount = int32(runtime.NumCPU())
	p := NewPool(-256)
	assert.Equal(t, p.maxThreads, expectedThreadCount)
}

func TestThreadPool_CorrectWorkerCount(t *testing.T) {
	const THREADS_COUNT = 16
	p := NewPool(THREADS_COUNT)
	assert.EqualValues(t, p.maxThreads, THREADS_COUNT)
}

func TestThreadPoo_NumberOfTasksEqualToWorkers(t *testing.T) {
	const THREADS_COUNT = 8
	p := NewPool(THREADS_COUNT)

	var makeSureThatAllTasksExecuted atomic.Int32
	var sleepDuration = 2000 * time.Millisecond

	for index := 0; index < THREADS_COUNT; index++ {
		p.SubmitTask(func() {
			makeSureThatAllTasksExecuted.Add(1)
			time.Sleep(sleepDuration)
		})
	}

	var start = time.Now()
	p.ProcessSubmittedTasks()
	var elapsed = time.Since(start)

	assert.EqualValues(t, makeSureThatAllTasksExecuted.Load(), THREADS_COUNT)
	assert.EqualValues(t, p.tasksSubmitted, THREADS_COUNT)
	assert.EqualValues(t, p.waitingTasksCount, 0)
	assert.Less(t, elapsed, 3000*time.Millisecond) // maybe 3000 is too strict, bump it to 4000 if the test starts to fail.
	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}

func TestThreadPoo_NumberOfTasksIsGreaterThenWorkerCount(t *testing.T) {
	const THREADS_COUNT = 4
	const TASKS_COUNT = 16
	p := NewPool(THREADS_COUNT)

	var makeSureThatAllTasksExecuted atomic.Int32
	var sleepDuration = 2000 * time.Millisecond

	for index := 0; index < TASKS_COUNT; index++ {
		p.SubmitTask(func() {
			makeSureThatAllTasksExecuted.Add(1)
			time.Sleep(sleepDuration)
		})
	}

	var start = time.Now()
	p.ProcessSubmittedTasks()
	var elapsed = time.Since(start)

	assert.EqualValues(t, makeSureThatAllTasksExecuted.Load(), TASKS_COUNT)
	assert.EqualValues(t, p.tasksSubmitted, TASKS_COUNT)
	assert.EqualValues(t, p.waitingTasksCount, 12)
	assert.Less(t, elapsed, 9000*time.Millisecond)
	assert.True(t, p.tasksQueue.Empty())
	assert.True(t, p.waitingQueue.Empty())
}
