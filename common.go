package main

// Given n KB, computes number of bytes.
func KiB(n int64) int64 {
	return n * 1024
}

func MiB(n int64) int64 {
	return KiB(n) * 1024
}

func GiB(n int64) int64 {
	return MiB(n) * 1024
}

func TiB(n int64) int64 {
	return GiB(n) * 1024
}
