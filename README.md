# Overall desciption
An implementation of a thread pool in `Go` programming language. 

## Inspirations
The idea behind this project was to get accustomed to `Go` and its concurrency model. 

Core structure: 
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
