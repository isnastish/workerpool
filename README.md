## Inspirations
The idea behind this project was to get accustomed to `Go` programming language and its concurrency model
by implementing a thread pool. Worth noting that there are multiple ways you can write a thread pool, 
and the way I did it, probably not how an experienced Golang programmer would do. 
My current solution is based around thread-safe queues, but that could be easily replaced with channels, 
the core mechanism for passing data between go routines and synchronizing them.

> **NOTE** This project was written exclusively for learning purposes and should never be used in a production. 

## Overall description
As mentioned above, a decision was made to use thread-safe queues for tasks submission and processing, 
though use of channels will be more natural. The core data type looks like this:
```go
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

	logsEnabled bool
	*Logger
}
```
Where `maxThreads` is the maximum number of goroutines running concurrently.
`waitingQueue` is used when all the workers (goroutines) are busy and no new can be spawned, a task is put into a waiting queue.
`submitQueue` is responsible for tasks submission.
`workQueue` a queue to pull work from.
The rest of the data are internals and easily understandable by looking at code.

All the logic is happening inside `processTasks()` function, which is itself is executed in a separate go routine.
This was mainly done to add a possibility to process tasks on the background while some more still could be submitted.

The flow is pretty straightforward, we pull tasks from a `submitQueue` and enqueue them into `workQueue`. 
Spawned workers constantly polling a work queue for available tasks and execute them. 
If the amount of workers is equal to `maxThreads` all the subsequent tasks are pushed into a `waitQueue` instead. 
No new workers are spawned until a wait queue is empty.

Example:
```go
const nThreads = 8
p := NewPool(nThreads)
for i := 0; i < (1 << 10); i++ {
	p.SubmitTask(func(){
		// do something
	})	
}
p.Wait()
```

> **IMPORTANT** Each call to `NewPool(...)` should be supplemented with `Wait()` after all the tasks have been submitted.

## Example
A simple web-crawler was implemented to demonstrate the functionality of a thread pool in action. 
An example could be found in `example.go` file. The programm traverses a specified url up to a certain depth (supplied from the command line)
in a breadth-first search fashion, and outputs all href(s) to stdout.

In orde to achieve that goal I had to implement a simple, generic stack with `Push/TryPop/Empty and Size` methods.
Here is the stack data type and its core function declarations:
```go
type Stack[T any] struct {
	count int
	data  []T
}

func (s *Stack[T]) Push(v T)
func (s *Stack[T]) TryPop(v *T) bool
func (s *Stack[T]) Empty() bool
func (s *Stack[T]) Size() int
```

Running the example: 
```sh
go build
./example -depth 3 -url https://golang.com
```

> **NOTE** For more examples look at the `thread_pool_test.go`, where I implemented filling in a giant (4GB) buffer of bytes concurrently
> and parallelized some sorting algorithms.

## Logging
zerolog is used as an underlying system for logging with custom settings to produce nicely formatted logs: 
> **EXAMPLE** 24 Mar 24 10:32 CET24 Mar 24 10:32 CET |DEBUG| Msg: worker finished CurrentThreads: 33