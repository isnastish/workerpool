package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	cli := MakeCli()
	cli.ParseArgs()

	if cli.GenFile {
		// TODO(alx): Log file generation as a progress bar.
		GenerateFile(cli.Filepath, cli.NumLines)
	}

	fd, err := os.Open(cli.Filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	o := MakeOrchestrator(fd, cli.ChunkSize, cli.Verbose)
	o.RegisterWorkerGroup(cli.NumWorkers)

	startTime := time.Now()

	o.Start()
	o.End()

	fmt.Printf("Task took: %s\n", time.Since(startTime))
}
