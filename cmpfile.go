// Copyright © 2018 Chad Netzer <chad.netzer@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package hardlinkable

import (
	"bytes"
	"io"
	"os"

	I "github.com/chadnetzer/hardlinkable/internal/inode"
)

func areFileContentsEqual(s status, pathname1, pathname2 string) (bool, error) {
	f1, openErr := os.Open(pathname1)
	if openErr != nil {
		return false, openErr
	}
	defer f1.Close()

	f2, openErr := os.Open(pathname2)
	if openErr != nil {
		return false, openErr
	}
	defer f2.Close()

	eq, err := fileContentsEqual(s, f1, f2)
	return eq, err
}

// Return true if f1 and f2 have identical contents. Otherwise return false.
func fileContentsEqual(s status, f1, f2 *os.File) (bool, error) {
	var atEnd bool
	bufSize := minCmpBufSize

	for {
		n1, err1 := I.ReadChunk(f1, s.cmpBuf1)
		n2, err2 := I.ReadChunk(f2, s.cmpBuf2)

		if n1 != n2 {
			return false, nil
		}

		if n1 > 0 {
			// If buf lengths are longer than what we read, re-slice to new
			// read length.
			if n1 < len(s.cmpBuf1) {
				s.cmpBuf1 = s.cmpBuf1[:n1]
				atEnd = true
			}
			if n2 < len(s.cmpBuf2) {
				s.cmpBuf2 = s.cmpBuf2[:n2]
				atEnd = true
			}

			eq := bytes.Equal(s.cmpBuf1, s.cmpBuf2)
			s.Results.addBytesCompared(uint64(n1 + n2))
			s.Progress.Show()
			if !eq {
				return false, nil
			}
		}

		// Process errors after processing the read bytes
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil
			} else if err1 == io.EOF && err2 != io.EOF {
				return false, err2
			} else {
				return false, err1
			}
		}
		// Re-slice buffer to increase length up to capacity.
		// Basically, start with a smaller buffer to reduce IO when files are
		// definitely unequal.  As files are found to be equal, increase the
		// buffer size, to speed up comparisons of large equal files.
		if !atEnd && bufSize < maxCmpBufSize {
			bufSize *= 2
			if bufSize > maxCmpBufSize {
				bufSize = maxCmpBufSize
			}
			s.cmpBuf1 = s.cmpBuf1[:bufSize]
			s.cmpBuf2 = s.cmpBuf2[:bufSize]
		}
	}
}
