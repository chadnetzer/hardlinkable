package inode

import (
	"fmt"
	"os"
)

// ReadChunk will retry Read() until it fills the buf, or reaches EOF or
// an error.
func ReadChunk(f *os.File, buf []byte) (n int, err error) {
	// For Posix reads of normal files, Read() will almost certainly return
	// a maximal Read() (or non-EOF error), but just in case, we make sure
	// to attempt to return a maximal chunk anyway.  Simple spin protection
	// in case of (0,nil) Read() returns, which again shouldn't happen.
	// Basically, this is pure overkill. :)
	const spinLimit = 10
	spinCount := 0
	N := len(buf)
	for {
		var nn int
		nn, err = f.Read(buf)
		n += nn
		if n == N || err != nil {
			break
		}

		if nn == 0 {
			spinCount++
		} else {
			spinCount = 0
			buf = buf[nn:] // crawl forward
		}
		if spinCount > spinLimit {
			return n, fmt.Errorf("stuck read for file: %v", f.Name())
		}
	}
	return
}
