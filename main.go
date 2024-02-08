package main

import (
	"fmt"
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

	o := MakeOrchestrator(cli.Filepath, cli.ChunkSize, cli.Verbose)
	o.RegisterWorkerGroup(cli.NumWorkers)

	o.Run()
}
