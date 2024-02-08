# Overall desciption
An attempt to write a worker pool in `Go` programming language. 
The idea behind it was to read contents of the file concurrently, chunk by chunk, 
where a chunk size is specified as a command-line argument using `-chunksize` flag.

The amount of workers to participate in reading the file simultaneously could be specified 
with `-workers` flag, the default value is `16`.

Among other flags are `-file`, used to specify a source file. If you wish a test file to be generated, `-genfile` option could be used together with `-numlines` to specify the amount of raws
in a file.

`-verbose` flag is used to display an intermediate state of each worker. 
For example: `worker 1, chunk [24343, 97979)`, where chunk is the amount of bytes read by the current worker.

For help use `-help` option.

# Core implementation.
The main data structure is `Orchestrator` responsible for distributing jobs between multiple workers.
Four the most important fields to pay attention to are `JobsQueue` which is a buffered channel for 
submitting jobs, `ResultsQueue` - buffered channel for receiving results from workers,
`WorkerPool` to keep track of all available workers, and `NumJobs` being total number of jobs.
The rest of the members are internal and shouldn't be exposed.

```Go
type Orchestrator struct {
	JobsQueue    chan Job
	ResultsQueue chan JobResult
	WorkerPool   map[int]*Worker
	NumJobs      int

	// These data is only related to work that we're doing.
	// So maybe it's better to keep it separate?
	// Take a look at C++ thread pool example on how to make it generic.
	Fd        *os.File
	FileSize  int64
	ChunkSize int64
	RemSize   int64

	Verbose bool
}
```

To register worker use `Orchestrator.RegisterWorker()` or, if multiple workers are desired to be registered, `Orchestrator.RegisterWorkerGroup()`.

```Go 
type Worker struct {
	Id      int
	Jobs    <-chan Job
	Results chan<- JobResult
}
```

Jobs quantity is computed automatically by the orchestrator based on supplied file's and chunk sizes.
For example, if file is equal to `8456` and a chunk size was specified as `4096`, three jobs will be created. 

```go
job0 := Job{Index: 0, Offset: 0, BytesToRead: 4096}
job1 := Job{Index: 1, Offset: 4096, BytesToRead: 4096}
job2 := Job{Index: 2, Offset: 8096, BytesToRead: filesize % chunkSize}
```

The amount of results in a `ResultsQueue` would be equivalent to the amount of jobs being submitted.

# Building the programm
On windows `build.bat` file should be executed which outputs a binary into `/build/` directory.
>**TODO** Write a make file to easily build it on Unix.

# Example
The following command would generate a test file and invoke `16` workers reading one chunk of `8096` at a time.

`build/workers.exe -genfile -chunksize 8096`

which gives the next output

```
Generating file: [.................................]
 Took: 6.7476638s

 Reading file: [.................................]
 Took: 107.9437ms
```