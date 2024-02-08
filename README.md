# Core data structures:

Cli
```go
type Cli struct {
	// Provide already existing file.
	Filepath string

	// Specify chunk size.
	ChunkSize int64

	// Specify number of workers to be involved in file processing.
	NumWorkers int

	// File to be generated.
	GenFile bool

	// Number of lines to generate.
	NumLines int64

	// Output intermediate states while reading the file.
	Verbose bool
}
```

Orchestrator
```go
type Orchestrator struct {
	JobsQueue    chan Job
	ResultsQueue chan JobResult
	WorkerPool   map[int]*Worker
	NumJobs      int64

	Fd        *os.File
	FileSize  int64
	ChunkSize int64

	Verbose bool
}
```

Worker
```go
type Worker struct {
	Id      int
	Jobs    <-chan Job
	Results chan<- JobResult
}
```

Chunk representation read from a file:
```go
type ReadChunk struct {
	Index     int64
	Offset    int64
	BytesRead int64
	Data      []byte
}
```

Submitted jobs
```go
type Job struct {
	Index       int64
	Offset      int64
	BytesToRead int64
}
```
