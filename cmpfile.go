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

package main

import (
	"bytes"
	"io"
	"os"
)

func areFileContentsEqual(pathname1, pathname2 string) (bool, uint64, error) {
	f1, openErr := os.Open(pathname1)
	if openErr != nil {
		return false, 0, openErr
	}
	defer f1.Close()

	f2, openErr := os.Open(pathname2)
	if openErr != nil {
		return false, 0, openErr
	}
	defer f2.Close()

	eq, bytes, err := cmpReaderContents(f1, f2)
	return eq, bytes, err
}

// Return true if r1 and r2 have identical contents. Otherwise return false.
func cmpReaderContents(r1, r2 io.Reader) (bool, uint64, error) {
	const bufSize = 8192
	buf1 := make([]byte, bufSize)
	buf2 := make([]byte, bufSize)
	var N uint64 // total bytes compared

	for {
		n1, err1 := r1.Read(buf1)
		_, err2 := r2.Read(buf2)
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, N, nil
			} else if err1 == io.EOF && err2 != io.EOF {
				return false, N, err2
			} else {
				return false, N, err1
			}
		}

		if !bytes.Equal(buf1, buf2) {
			return false, N, nil
		}
		N += uint64(n1)
	}
	return false, N, nil // never reached
}
