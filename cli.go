package main

import "flag"

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

func MakeCli() *Cli {
	var cli = new(Cli)

	var defaultChunkSize int64 = 16384
	var defaultNumWorkers int = 16
	var defaultNumLines int64 = 1 << 20
	var defaultFileName string = "generated_large.go"

	flag.StringVar(&cli.Filepath, "file", defaultFileName, "Full path to file to be read.")
	flag.Int64Var(&cli.ChunkSize, "chunk_size", defaultChunkSize, "Chunk size to be read by a single worker.")
	flag.IntVar(&cli.NumWorkers, "workers", defaultNumWorkers, "Number of workers to participate in concurrent file read.")
	flag.BoolVar(&cli.GenFile, "genfile", false, "File to be generated.")
	flag.Int64Var(&cli.NumLines, "numlines", defaultNumLines, "Number of lines in file.")
	flag.BoolVar(&cli.Verbose, "verbose", false, "Output intermediate states while reading the file.")

	return cli
}

func (cli *Cli) ParseArgs() {
	flag.Parse()
}
