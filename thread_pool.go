package main

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type ThreadFunc func()

type Metrics struct {
	tasksSubmitted   uint32
	tasksDone        uint32
	tasksQueued      uint32
	routinesSpawned  uint32
	routinesFinished uint32
}

type ThreadPool struct {
	maxThreads uint32

	submitQueue  *Queue[ThreadFunc]
	waitingQueue *Queue[ThreadFunc]
	workQueue    *Queue[ThreadFunc]

	wg          sync.WaitGroup
	doneCh      chan struct{}
	threadCount uint32

	metrics Metrics

	waiting int32

	blocked bool

	// NOTE: logsEnabled flag should be removed once I figure out how to do concurrent logging.
	// Because currently, with logging enabled, some tests would block forewer due to the fact
	// that the writer is not protected a mutex and prohibits simultaneous writes.
	// Sometimes all the logs could be displayed correctly without blocking, but sometimes they don't.
	logsEnabled bool
	*Logger
}

func NewPool(numThreads ...uint32) *ThreadPool {
	// Get a number of cores usable by the current process.
	// This is equivalent to maximum amount of goroutines (workers) created.
	hardwareCPU := uint32(runtime.NumCPU())

	var maxThreads uint32
	if len(numThreads) > 0 {
		if numThreads[0] < 1 || numThreads[0] > hardwareCPU {
			maxThreads = hardwareCPU
		} else {
			maxThreads = numThreads[0]
		}
	} else {
		maxThreads = hardwareCPU
	}

	p := &ThreadPool{
		maxThreads:   maxThreads,
		submitQueue:  NewQueue[ThreadFunc](),
		workQueue:    NewQueue[ThreadFunc](),
		waitingQueue: NewQueue[ThreadFunc](),
		wg:           sync.WaitGroup{},
		doneCh:       make(chan struct{}),
		Logger:       NewLogger("debug"),

		// TODO: Uncomment this line once the logging is thread-safe
		// logsEnabled: true,
	}

	go p.processTasks()

	return p
}

func (p *ThreadPool) SubmitTask(task func()) {
	if nil == task {
		if p.logsEnabled {
			p.logger.Info().Msg("nil task was submitted")
		}
		return
	}

	if p.blocked {
		if p.logsEnabled {
			p.logger.Info().Msg("thread pool blocked, no more tasks could be submitted")
		}
		return
	}

	if p.logsEnabled {
		p.logger.Info().Msg("task has been submitted")
	}

	p.submitQueue.Push(task)
	atomic.AddUint32(&p.metrics.tasksSubmitted, 1)
}

func (p *ThreadPool) processTasks() {
	var running bool = true
	for running {
		// Firstly, process all the tasks from the waiting queue until it is empty.
		if !p.waitingQueue.Empty() {
			var wTask ThreadFunc
			for p.waitingQueue.TryPop(&wTask) {
				p.workQueue.Push(wTask)

				var sTask ThreadFunc
				if p.submitQueue.TryPop(&sTask) {
					p.waitingQueue.Push(sTask)
				}
			}
			continue
		}

		var task ThreadFunc
		if p.submitQueue.TryPop(&task) {
			// New workers can be spawned only if we haven't reached the limit of maximum workers,
			// or we've reached the limit but then some of them finished their work, in that case
			// new could be created.
			if atomic.LoadUint32(&p.threadCount) < p.maxThreads {
				p.workQueue.Push(task)

				if p.logsEnabled {
					p.logger.Info().Msg("worker created")
				}

				p.wg.Add(1)
				go p.worker()

				p.metrics.routinesSpawned++
			} else {
				// If all the workers are busy, put task into a waiting queue for further processing.
				if p.logsEnabled {
					p.logger.Info().Msg("all workers are busy, task is pushed to the waiting queue")
				}

				p.waitingQueue.Push(task)
				p.metrics.tasksQueued++
			}
		} else {
			if atomic.LoadInt32(&p.waiting) != 0 {
				running = false
			}
		}
	}

	// Wait for all spawned workers to finish their work.
	p.wg.Wait()

	// Notify Wait() procedure that the channel was closed.
	close(p.doneCh)
}

func (p *ThreadPool) Debug_GetMetrics() Metrics {
	return p.metrics
}

func (p *ThreadPool) worker() {
	if p.logsEnabled {
		p.logger.Info().Msg("worker started")
	}

	defer func() {
		if p.logsEnabled {
			p.logger.Info().Msg("worker finished")
		}
		p.wg.Done()
	}()

	atomic.AddUint32(&p.threadCount, 1)

	var task ThreadFunc
	for !p.workQueue.Empty() {
		if p.workQueue.TryPop(&task) {
			atomic.AddUint32(&p.metrics.tasksDone, 1)
			task()
		}
	}

	// Decrement threads count so other workers can be spawned,
	// in case the waiting queue is not empty and waiting for at least one worker to complete.
	atomic.AddUint32(&p.threadCount, ^uint32(0))
	atomic.AddUint32(&p.metrics.routinesFinished, 1)
}

func (p *ThreadPool) Wait() {
	// No more tasks could be submitted
	p.blocked = true

	// Put the pool in a waiting state.
	// That implies that all the earlier submitted tasks should run until their completion.
	atomic.AddInt32(&p.waiting, 1)

	// Wait for all remaining tasks to complete. Shut down the pool
	<-p.doneCh
}
