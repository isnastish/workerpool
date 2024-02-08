package main

// TODO(alx): Use workers to generate file.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// Source set of characters for random string generation.
var charPool = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345689!@$^&*()_+")

// Seed random generator with current time value.
func Init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// Compute checkSum for the given string.
func computeSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	checkSum := h.Sum(nil)
	return hex.EncodeToString(checkSum)
}

// Populate file with generated contents.
func GenerateFile(filepath string, numLines int64) {
	Init()

	var (
		buf           = make([]rune, len(charPool))
		i      int64  = 0
		header string = `
/* This file was generated. Don't modify it manually.\n
In order to regenerate it run ./workers -genfile <filename>.
This file shouldn't be included into a build.\n\n*/
`
	)

	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	terminateCh := make(chan struct{}, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		DisplayProgressBar(terminateCh)
		close(terminateCh)
		wg.Done()
	}()

	file.WriteString(header)
	file.WriteString("package main\n\nvar (\n")

	for ; i < numLines; i++ {
		for k := 0; k < len(charPool); k++ {
			index := rand.Intn(len(charPool))
			buf[k] = charPool[index]
		}
		str := computeSHA256(string((buf)))
		fmt.Fprintf(file, "\thash%d = []rune(\"%s\")\n", i, str)
	}
	file.WriteString(")\n")

	// Signal goroutine to stop displaying progress bar.
	terminateCh <- struct{}{}

	wg.Wait()
}

// Display progress bar while file is being generated.
func DisplayProgressBar(terminateCh chan struct{}) {
	fmt.Print("Generating file: [")
	for {
		select {
		case <-terminateCh: // When received terminate event
			fmt.Print("]\n")
			return
		default:
			fmt.Print(".")
			time.Sleep(200 * time.Millisecond)
		}
	}
}
