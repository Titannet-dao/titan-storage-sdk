package storage

import "io"

// ProgressReader is a wrapper around an io.Reader that reports progress as data is read.
type ProgressReader struct {
	io.Reader               // The underlying io.Reader to read data from.
	Reporter  func(r int64) // A function to report progress, taking the number of bytes read as an argument.
}

// Read reads data from the underlying io.Reader and reports progress using the provided Reporter function.=
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.Reporter(int64(n))
	return
}
