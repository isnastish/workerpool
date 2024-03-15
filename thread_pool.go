package main

// What we should do instead is
import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	// TODO: Remove zap logger, it's an overkill for this project.
	// Probably too heavy to be used in this project
	"go.uber.org/zap"
)

type Metrics struct {
	tasksSubmitted  uint32
	tasksDone       uint32
	tasksQueued     uint32
	threadsSpawned  uint32
	threadsFinished uint32
}

type ThreadPool struct {
	maxThreads     uint32
	waitingQueue   *ThreadSafeQueue[func()]
	tasksQueue     *ThreadSafeQueue[func()]
	metrics        Metrics
	displayMetrics bool
	wg             sync.WaitGroup
	threadCount    uint32
	zapLogger      *zap.Logger

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
		waitingQueue:   NewQueue[func()](threadSafe),
		tasksQueue:     NewQueue[func()](threadSafe),
		maxThreads:     maxThreads,
		displayMetrics: displayMetrics,
		wg:             sync.WaitGroup{},
		zapLogger:      logger,
	}

	return p
}

func (p *ThreadPool) SubmitTask(task func()) {
	if nil == task {
		p.zapLogger.Warn("nil task was submitted. Rejecting.")
		return
	}

	// Submit tasks into a tasksQueue for later execution by workers.
	p.tasksQueue.Push(task)
	p.metrics.tasksSubmitted++
}

// Rename to Wait function.
// If ProcessSubmittedTasks is executed one more time on an empty queue, we end up in a trouble
// because p.zapLogger resource was released. So most probably we have to prohibit reuse of thread pool
func (p *ThreadPool) ProcessSubmittedTasks() {
	defer p.zapLogger.Sync()

	p.running = true

	if p.tasksQueue.Empty() {
		p.zapLogger.Info("No tasks submitted, exiting.")
		return
	}

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
			if atomic.LoadUint32(&p.threadCount) < p.maxThreads {
				p.spawnWorker(task)
			} else {
				p.waitingQueue.Push(task)
				// This is only indicative since the same element will be pushed over and over again
				// if all the workers are busy.
				// p.metrics.tasksQueued++
			}
		} else {
			if p.waitingQueue.TryPop(&task) {
				// Check if there are workers available,
				// if not, put the task back into a waiting queue.
				if atomic.LoadUint32(&p.threadCount) < p.maxThreads {
					p.spawnWorker(task)
				} else {
					// Sleep for half a second if all the workers are busy,
					// before pushing a task back to the waiting queue.
					time.Sleep(500 * time.Millisecond)
					p.waitingQueue.Push(task)
					p.metrics.tasksQueued++
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

	// Display accumulated metrics.
	if p.displayMetrics {
		p.zapLogger.Info(
			"Metrics",
			zap.Uint32("Tasks submitted", p.metrics.tasksSubmitted),
			zap.Uint32("Tasks done", p.metrics.tasksDone),
			zap.Uint32("Tasks queued", p.metrics.tasksQueued),
			zap.Uint32("Threads spawned", p.metrics.threadsSpawned),
			zap.Uint32("Threads finished", p.metrics.threadsFinished),
		)
	}
}

func (p *ThreadPool) GetMetrics() Metrics {
	return p.metrics
}

func (p *ThreadPool) spawnWorker(task func()) {
	// Create a worker and assign a task for it to execute.
	p.wg.Add(1)
	go p.worker(task)
	atomic.AddUint32(&p.threadCount, 1)
	p.metrics.threadsSpawned++
}

func (p *ThreadPool) worker(task func()) {
	for task != nil {
		task()
		atomic.AddUint32(&p.metrics.tasksDone, 1)
		if !p.tasksQueue.TryPop(&task) {
			task = nil
		}
	}

	// Decrement threads count so other workers can be spawned,
	// in case the waiting queue is not empty and waiting for at least one worker to complete.
	atomic.AddUint32(&p.threadCount, ^uint32(0))
	atomic.AddUint32(&p.metrics.threadsFinished, 1)

	p.wg.Done()
}
