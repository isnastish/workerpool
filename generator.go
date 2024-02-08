package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
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
		i      int64  = 0
		buf           = make([]rune, len(charPool))
		header string = "/* This file was generated. Don't modify it manually.\n In order to regenerate it run -genfile <filename>.*/\n\n"
	)

	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	file.WriteString(header)

	fmt.Fprintf(file, "package main\n\nvar (\n")
	for ; i < numLines; i++ {
		for k := 0; k < len(charPool); k++ {
			index := rand.Intn(len(charPool))
			buf[k] = charPool[index]
		}
		str := computeSHA256(string((buf)))
		fmt.Fprintf(file, "\thash%d = []rune(\"%s\")\n", i, str)
	}
	fmt.Fprint(file, ")")
}
