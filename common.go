package main

import (
	"fmt"
	"time"
)

// Given n(KB) returns number of bytes.
func KiB(n int64) int64 {
	return n * 1024
}

// Given n(MB) returns number of bytes.
func MiB(n int64) int64 {
	return KiB(n) * 1024
}

// Given n(GB) returns number of bytes.
func GiB(n int64) int64 {
	return MiB(n) * 1024
}

// Given n(TB) returns number of bytes.
func TiB(n int64) int64 {
	return GiB(n) * 1024
}

// Displays progress bar.
func DisplayProgressBar(msg string, sleepTimeMs int, symbol rune, terminateCh chan struct{}) {
	fmt.Printf("%s: [", msg)
	for {
		select {
		case <-terminateCh:
			fmt.Print("]\n")
			return
		default:
			fmt.Printf("%s", string(symbol))
			time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
		}
	}
}
