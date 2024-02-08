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
		genStartTime := time.Now()
		GenerateFile(cli.Filepath, cli.NumLines)
		fmt.Printf("Took: %s\n\n", time.Since(genStartTime))
	}

	fd, err := os.Open(cli.Filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	o := MakeOrchestrator(fd, cli.ChunkSize, cli.Verbose)
	o.RegisterWorkerGroup(cli.NumWorkers)

	fmt.Println("Reading file...")

	startTime := time.Now()
	o.Start()
	o.End()
	fmt.Printf("Took: %s\n", time.Since(startTime))
}
