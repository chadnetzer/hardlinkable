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

	eq, err := readerContentsEqual(s, f1, f2)
	return eq, err
}

// Return true if r1 and r2 have identical contents. Otherwise return false.
func readerContentsEqual(s status, r1, r2 io.Reader) (bool, error) {
	const bufSize = 8192
	buf1 := make([]byte, bufSize)
	buf2 := make([]byte, bufSize)

	for {
		n1, err1 := r1.Read(buf1)
		n2, err2 := r2.Read(buf2)
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil
			} else if err1 == io.EOF && err2 != io.EOF {
				return false, err2
			} else {
				return false, err1
			}
		}

		if !bytes.Equal(buf1, buf2) {
			return false, nil
		}
		s.Results.addBytesCompared(uint64(n1 + n2))
		s.Progress.Show()
	}
}
