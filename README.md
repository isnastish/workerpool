## Inspirations
The idea behind this project was to get accustomed to `Go` programming language and its concurrency model
by implementing a thread pool. Worth noting that there are multiple ways you can write a thread pool, 
and the way I did it, probably not how an experienced Golang programmer would do. 
My current solution is based around thread-safe queues, but that could be easily replaced with channels, 
which is the core core mechanism for passing data between go routines and synchronizing them.

> **NOTE** This project was written exclusively for learning purposes and should never be used in production. 

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
A simple web-crawler was implemented to demonstrate the functionality of a thread pool in action. 
An example could be found in `example.go` file. The programm traverses a specified url up to a certain depth (supplied from the command line)
in a breadth-first search fashion, and outputs all href(s) to stdout.

In orde to achieve that goal I had to implement a simple, generic stack with `Push/TryPop/Empty and Size` methods.
Here is the stack data type and its core functions declarations:
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

```shell
go build
./exmple -depth 3 -url https://golang.com
```
