package main

/*
NOTE(alx): Currently we distribute work across multiple workers based on file size
that we want to read chunk by chunk. But it requires additional code and becomes convoluted.
A better approach would be, if we could synchronize workers' work with a help of orchestrator,
so they read the file chunk by chunk until one of them reaches EOF.
Then an orchestrator should accumulate the result and send it over the network.

*The process of determining of how many chunks to read, offsets, and other things should be better structured.
TODO(alx):

[ ] Try with unbuffered channels.
[ ] Implement the other way around, writing chunks again to the file.
[ ] Write an API for workers so we have a clear separation between its core functionality
    and all the parameters/options that it requires to function properly.
*/

import (
	"fmt"
	"io"
	"log"
	"os"
)

type ReadChunk struct {
	Index     int64 // TODO(alx): Replace with int32 or int.
	Offset    int64
	BytesRead int64
	Data      []byte
}

type Job struct {
	Index       int64 // TODO(alx): Replace with plain integer.
	Offset      int64
	BytesToRead int64
}

type JobResult struct {
	Chunk ReadChunk
}

type Worker struct {
	Id      int
	Jobs    <-chan Job
	Results chan<- JobResult
}

type Orchestrator struct {
	JobsQueue    chan Job
	ResultsQueue chan JobResult
	WorkerPool   map[int]*Worker
	NumJobs      int64

	// These things should either be removed or encapsulated better.
	Fd        *os.File
	FileSize  int64
	ChunkSize int64

	Verbose bool
}

func MakeJob(index, offset, bytesToRead int64) *Job {
	return &Job{
		Index:       index,
		Offset:      offset,
		BytesToRead: bytesToRead,
	}
}

func MakeWorker(id int, jobs <-chan Job, results chan<- JobResult) *Worker {
	return &Worker{
		Id:      id,
		Jobs:    jobs,
		Results: results,
	}
}

// Or channels can be passed here.
// And maybe file should be included into Job struct
func (w *Worker) DoWork(fd *os.File, verbose bool) {
	for job := range w.Jobs {
		var (
			startByte = job.Offset
			endByte   = startByte + job.BytesToRead
		)

		if verbose {
			str := fmt.Sprintf(
				"Worker %d, chunk: [%d, %d), bytes: %d\n",
				w.Id,
				startByte,
				endByte,
				job.BytesToRead,
			)

			log.Println(str)
		}

		storage := make([]byte, job.BytesToRead)
		bytesRead, err := fd.ReadAt(storage, job.Offset)

		w.Results <- JobResult{
			Chunk: ReadChunk{
				Index:     job.Index,
				Offset:    job.Offset,
				BytesRead: int64(bytesRead),
				Data:      storage,
			},
		}

		if err == io.EOF {
			return
		} else if err != nil {
			log.Fatal(err)
		}

	}
}

// Maybe jobs count should be moved into a different function?
func MakeOrchestrator(fd *os.File, chunkSize int64, verbose bool) *Orchestrator {
	var (
		info, _           = fd.Stat()
		fileSize          = info.Size()
		chunksCount       = fileSize / chunkSize
		remSize           = fileSize % chunkSize
		oneJob      int64 = 0
	)

	if remSize != 0 {
		oneJob = 1
	}

	// NOTE(alx): This is extremely important to have buffered channels instead of
	// unbuffered. Those act like a queue of elements.
	return &Orchestrator{
		JobsQueue:    make(chan Job, chunksCount+oneJob),
		ResultsQueue: make(chan JobResult, chunksCount+oneJob),
		WorkerPool:   make(map[int]*Worker),
		NumJobs:      chunksCount,
		Fd:           fd,
		FileSize:     fileSize,
		ChunkSize:    chunkSize,
		Verbose:      verbose,
	}
}

func (o *Orchestrator) Start() {
	var (
		remSize        = o.FileSize % o.ChunkSize
		offset   int64 = 0
		jobIndex int64 = 0
	)

	// Spin up registered workers.
	for _, w := range o.WorkerPool {
		go w.DoWork(o.Fd, o.Verbose)
	}

	for ; jobIndex < o.NumJobs; jobIndex++ {
		// Don't allocate memory which you don't use!
		o.JobsQueue <- *MakeJob(jobIndex, offset, o.ChunkSize)
		offset += o.ChunkSize
	}

	if remSize != 0 {
		o.JobsQueue <- *MakeJob(jobIndex, offset, remSize)
	}

	close(o.JobsQueue)
}

func (o *Orchestrator) End() {
	// accumulate chunks.
	readChunks := make([]ReadChunk, int(o.NumJobs)+1)

	for i := 0; i < int(o.NumJobs)+1; i++ {
		jobRes := <-o.ResultsQueue
		readChunks = append(readChunks, jobRes.Chunk)
	}

	if o.Verbose {
		// TODO(alx): Write the result into a file following the same approach.
		log.Println("File processing finished.")
	}
}

func (o *Orchestrator) RegisterWorker(id int, w *Worker) {
	if o.WorkerPool != nil {
		o.WorkerPool[id] = w
	}
}

func (o *Orchestrator) RegisterWorkerGroup(numWorkers int) {
	for workerId := 0; workerId < numWorkers; workerId++ {
		o.RegisterWorker(workerId, MakeWorker(workerId, o.JobsQueue, o.ResultsQueue))
	}
}
