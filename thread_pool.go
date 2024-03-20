package main

// On pool creation we should spawn a single go routine which will process all
// incoming tasks on the background.
// p.Wait() procedure should wait for all submitted tasks to complete,
// what's important is that no more tasks could be submitted after p.Wait() has been invoked.
// But those submitted already should run until their completion.
// We should introduce separate queues in order to avoid queue contention.
// For example tasks are submitted to one queue (tasksQueue),
// but wokers pull out work from a different queue (poolWorkQueue/sharedWorkQueue).
// So we don't push and pull from the same queue.

// TODO: Write a mechanism which blocks tasks submission after Wait() function was invoked.
// Probably it should send some signal over the channel, or set a variable that we cannot accept new
// tasks anymore AND that we have to wait for all the earlier submitted tasks to complete.
// So we don't use doneCh when spawning a separate go routine with processTasks() procedure.

import (
	_ "fmt"
	"go.uber.org/zap"
	"runtime"
	"sync"
	"sync/atomic"
	_ "time"
)

type ThreadFunc func()

// Used for accumulating the metrics.
type Metrics struct {
	tasksSubmitted   uint32
	tasksDone        uint32
	tasksQueued      uint32
	routinesSpawned  uint32
	routinesFinished uint32
}

type ThreadPool struct {
	maxThreads     uint32
	waitingQueue   *Queue[ThreadFunc]
	submitQueue    *Queue[ThreadFunc]
	workQueue      *Queue[ThreadFunc]
	metrics        Metrics
	displayMetrics bool
	wg             sync.WaitGroup
	doneCh         chan struct{}
	threadCount    uint32
	zapLogger      *zap.Logger

	// atomic
	waiting int32

	blocked bool
	running bool
}

func NewPool(displayMetrics bool, numThreads ...uint32) *ThreadPool {
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

	logger, _ := zap.NewProduction()

	const threadSafe = true
	p := &ThreadPool{
		maxThreads:     maxThreads,
		submitQueue:    NewQueue[ThreadFunc](threadSafe),
		workQueue:      NewQueue[ThreadFunc](threadSafe),
		waitingQueue:   NewQueue[ThreadFunc](threadSafe),
		displayMetrics: displayMetrics,
		wg:             sync.WaitGroup{},
		doneCh:         make(chan struct{}),
		zapLogger:      logger,
	}

	go p.processTasks()

	return p
}

func (p *ThreadPool) SubmitTask(task func()) {
	if nil == task {
		p.zapLogger.Warn("nil task was submitted. Rejecting.")
		return
	}

	if p.blocked {
		p.zapLogger.Warn("Thread pool finished, no more tasks could be submitted.")
		return
	}

	// Submit tasks into a tasksQueue for later execution by workers.
	p.submitQueue.Push(task)
	atomic.AddUint32(&p.metrics.tasksSubmitted, 1)
}

func (p *ThreadPool) processTasks() {
	defer p.zapLogger.Sync()

	p.running = true

	// if p.tasksQueue.Empty() {
	// 	p.zapLogger.Info("No tasks submitted, exiting.")
	// 	return
	// }

	for p.running {
		// Pop tasks from a tasksQueue, if the amount of spawned goroutines less than
		// maxThreads, created a new worker passing a task to it.
		// Otherwise put in into a waiting queue.
		// An additional queue was only used for clarity and convenience,
		// the same result could be achieved by using a single queue and pushing elements to the back,
		// if all the workers are busy.

		// process all the tasks in a waiting queue
		if !p.waitingQueue.Empty() {
			var task ThreadFunc
			for p.waitingQueue.TryPop(&task) {
				p.workQueue.Push(task)

				var nextTask ThreadFunc
				if p.submitQueue.TryPop(&nextTask) {
					p.waitingQueue.Push(nextTask)
				}
			}
			continue
		}

		// TODO: Introduce a timeout when no tasks were submitted for some period of time (let's say 2000ml)
		// So we can terminate the loop and shut down gracefully this thread.

		var task ThreadFunc
		if p.submitQueue.TryPop(&task) {
			// If we exceeded the amount of all available workers,
			// put task into a waiting queue for further processing.
			if atomic.LoadUint32(&p.threadCount) < p.maxThreads {
				p.workQueue.Push(task)

				p.wg.Add(1)
				go p.worker()

				atomic.AddUint32(&p.threadCount, 1)
				p.metrics.routinesSpawned++
			} else {
				p.waitingQueue.Push(task)
				p.metrics.tasksQueued++
			}
		} else {
			// At this point the submitted queue is empty and a waiting queue should be empty as well.
			if atomic.LoadInt32(&p.waiting) != 0 {
				p.running = false
			}
		}

		//  else {
		// 	if p.waitingQueue.TryPop(&task) {
		// 		// Check if there are workers available,
		// 		// if not, put the task back into a waiting queue.
		// 		if atomic.LoadUint32(&p.threadCount) < p.maxThreads {
		// 			p.wg.Add(1)
		// 			go p.worker(task)
		// 			atomic.AddUint32(&p.threadCount, 1)
		// 			p.metrics.routinesSpawned++
		// 			// p.spawnWorker(task)
		// 		} else {
		// 			// Sleep for half a second if all the workers are busy,
		// 			// before pushing a task back to the waiting queue.
		// 			time.Sleep(500 * time.Millisecond)
		// 			p.waitingQueue.Push(task)
		// 			p.metrics.tasksQueued++
		// 		}
		// 	} else {
		// 		// Both tasks and waiting queues are empty
		// 		// Break out of the for loop.
		// 		p.running = false
		// 	}
		// }
	}

	// Wait for all spawned workers to finish their work.
	p.wg.Wait()

	// Display accumulated metrics.
	if p.displayMetrics {
		p.zapLogger.Info(
			"Metrics",
			zap.Uint32("Tasks submitted", atomic.LoadUint32(&p.metrics.tasksSubmitted)),
			zap.Uint32("Tasks done", p.metrics.tasksDone),
			zap.Uint32("Tasks queued", p.metrics.tasksQueued),
			zap.Uint32("Threads spawned", p.metrics.routinesSpawned),
			zap.Uint32("Threads finished", p.metrics.routinesFinished),
		)
	}

	p.doneCh <- struct{}{} // not required since close(p.doneCh) will do the job.
	close(p.doneCh)
}

func (p *ThreadPool) GetMetrics() Metrics {
	return p.metrics
}

func (p *ThreadPool) worker() {
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

	p.wg.Done()
}

func (p *ThreadPool) Wait() {
	p.blocked = true
	atomic.AddInt32(&p.waiting, 1)
	<-p.doneCh
}
