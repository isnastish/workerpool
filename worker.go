package main

/*
NOTE(alx): Currently we distribute work across multiple workers based on file size
that we want to read chunk by chunk. But it requires additional code and becomes convoluted.
A better approach would be, if we could synchronize workers' work with a help of orchestrator,
so they read the file chunk by chunk until one of them reaches EOF.
Then an orchestrator should accumulate the result and send it over the network.

[ ] Write an API for workers so we have a clear separation between its core functionality
    and all the parameters/options that it requires to function properly.
*/

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type ReadChunk struct {
	Index     int
	Offset    int64
	BytesRead int64
	Data      []byte
}

type Job struct {
	Index       int
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

func MakeWorker(id int, jobs <-chan Job, results chan<- JobResult) *Worker {
	return &Worker{
		Id:      id,
		Jobs:    jobs,
		Results: results,
	}
}

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

func MakeOrchestrator(filepath string, chunkSize int64, verbose bool) *Orchestrator {
	fd, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	var (
		info, _     = fd.Stat()
		fileSize    = info.Size()
		chunksCount = int(fileSize / chunkSize)
		remSize     = fileSize % chunkSize
		auxJOB      = 0
	)

	// If filesize % chunkSize ras a remainder, we would have to submit an additional job
	// to handle that tail.
	if remSize != 0 {
		auxJOB += 1
	}

	return &Orchestrator{
		JobsQueue:    make(chan Job, chunksCount+auxJOB),
		ResultsQueue: make(chan JobResult, chunksCount+auxJOB),
		WorkerPool:   make(map[int]*Worker),
		NumJobs:      chunksCount,
		Fd:           fd,
		FileSize:     fileSize,
		ChunkSize:    chunkSize,
		Verbose:      verbose,
		RemSize:      remSize,
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

func (o *Orchestrator) submitJob(index int, offset int64, bytesToRead int64) {
	o.JobsQueue <- Job{
		Index:       index,
		Offset:      offset,
		BytesToRead: bytesToRead,
	}
}

func (o *Orchestrator) Run() {
	var (
		terminateCh = make(chan struct{})
		wg          = sync.WaitGroup{}
		startTime   time.Time
	)

	// Just for displaying progress bar
	wg.Add(1)
	go func() {
		DisplayProgressBar("Reading file", 15, '#', terminateCh)
		close(terminateCh)
		wg.Done()
	}()

	startTime = time.Now()

	// Spin up registered workers.
	for _, w := range o.WorkerPool {
		go w.DoWork(o.Fd, o.Verbose)
	}

	var offset int64 = 0
	for jobIndex := 0; jobIndex < o.NumJobs; jobIndex++ {
		if jobIndex == o.NumJobs-1 && o.RemSize != 0 {
			o.submitJob(jobIndex, offset, o.RemSize)
			break
		}
		o.submitJob(jobIndex, offset, o.ChunkSize)
	}

	close(o.JobsQueue)

	for jobIndex := 0; jobIndex < o.NumJobs; jobIndex++ {
		<-o.ResultsQueue
	}

	// Send a signal to terminate progress bar.
	terminateCh <- struct{}{}

	// Wait for progress bar go routine to complete.
	wg.Wait()

	fmt.Println("Took: ", time.Since(startTime))

	if o.Verbose {
		log.Println("File processing finished.")
	}

	// Close file
	o.Fd.Close()
}
