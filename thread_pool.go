package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

type ThreadPool struct {
	waitingQueue      *ThreadSafeQueue[func()]
	tasksQueue        *ThreadSafeQueue[func()]
	threadCount       atomic.Int32 // atomic since will be accessed by multiple threads (workers).
	maxThreads        int32
	running           bool
	tasksSubmitted    int32 // metrics
	waitingTasksCount int32 // metrics

	wg sync.WaitGroup
}

func NewPool(numThreads int32) *ThreadPool {
	// Get a number of cores usable by the current process.
	// This is equivalent to maximum amount of goroutines (workers) created.
	hardwareCPU := int32(runtime.NumCPU())

	// If the number of threads less than 1 or greater than hardwareCPU, perform clipping.
	if numThreads < 1 {
		numThreads = 1
	} else if numThreads > hardwareCPU {
		numThreads = hardwareCPU
	}

	p := &ThreadPool{
		waitingQueue: NewQueue[func()](),
		tasksQueue:   NewQueue[func()](),
		maxThreads:   numThreads,
		wg:           sync.WaitGroup{},
	}

	return p
}

func (p *ThreadPool) SubmitTask(task func()) {
	if nil == task {
		fmt.Println("WARNING: nil task was submitted.")
		return
	}

	// Submit tasks into a tasksQueue for later execution by workers.
	p.tasksQueue.Push(task)
	p.tasksSubmitted++
}

func (p *ThreadPool) ProcessSubmittedTasks() {
	p.running = true

	for p.running {
		// Pop tasks from a tasksQueue, if the amount of spawned goroutines less than
		// maxThreads, created a new worker passing a task to it.
		// Otherwise put in into a waiting queue.
		// An additional queue was only used for clarity and convenience,
		// the same result could be achieved by using a single queue and pushing elements to the back,
		// if all the workers are busy.

		var task func()
		if p.tasksQueue.TryPop(&task) {
			// If we exceeded the amount of all available workers,
			// put task into a waiting queue for further processing.
			if p.threadCount.Load() < p.maxThreads {
				p.wg.Add(1)
				go p.worker(task)
				p.threadCount.Add(1)
			} else {
				p.waitingQueue.Push(task)
				p.waitingTasksCount++
			}
		} else {
			if p.waitingQueue.TryPop(&task) {
				// Check if there are workers available,
				// if not, put the task back into a waiting queue.
				if p.threadCount.Load() < p.maxThreads {
					p.wg.Add(1)
					go p.worker(task)
					p.threadCount.Add(1)
				} else {
					p.waitingQueue.Push(task)
				}
			} else {
				// Both tasks and waiting queues are empty
				// Break out of the for loop.
				p.running = false
			}
		}
	}

	// Wait for all spawned workers to finish their work.
	p.wg.Wait()
}

func (p *ThreadPool) worker(task func()) {
	for task != nil {
		task()
		if !p.tasksQueue.TryPop(&task) {
			task = nil
		}
	}
	// Decrement threads count so other workers can be spawned,
	// in case the waiting queue is not empty and waiting for at least one worker to complete.
	p.threadCount.Add(-1)
	p.wg.Done()
}
