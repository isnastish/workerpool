## Inspirations
The idea behind this project was to get accustomed to `Go` programming language and its concurrency model
by implementing a thread pool.

> **NOTE** This project was written exclusively for learning purposes and should never be used in real production. 

## Overall desciption

## Core data types and functionality

```go
type ThreadPool struct {
	maxThreads uint32

	waitingQueue *Queue[ThreadFunc]
	submitQueue  *Queue[ThreadFunc]
	workQueue    *Queue[ThreadFunc]

	wg          sync.WaitGroup
	doneCh      chan struct{}
	threadCount uint32

	metrics Metrics

	waiting int32

	blocked bool
}
```

## Example
