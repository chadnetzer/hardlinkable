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
	"os"
	"testing"
)

func initDifferentBufs(t *testing.T, b1, b2 []byte) {
	if cap(b1) != cap(b2) {
		t.Fatalf("different capacities for byte buffers: %v %v", cap(b1), cap(b2))
	}
	// reslice to full capacity, and initialize byte buffers with different
	// content
	b1 = b1[:cap(b1)]
	b2 = b2[:cap(b2)]
	for i := 0; i < cap(b1); i++ {
		b1[i] = 1
		b2[i] = 2
	}
}

// simple, inefficient repeated string maker
func makeString(s string, length int) string {
	for len(s) < length {
		s += s
	}
	return s[:length]
}

func TestFileContentComparison(t *testing.T) {
	topdir := setUp("Cmp", t)
	defer os.RemoveAll(topdir)

	ls := newLinkableState(&Options{})
	s := ls.status
	s.Progress = &disabledProgress{}

	var tests = []struct {
		content       [2]string
		wants         bool
		bytesCompared uint64
		errStr        string
	}{
		{[2]string{"", ""}, true, 0, "Zero length cmpContents() compared unequal"},
		{[2]string{"A", "A"}, true, 2, "Equal length 1 cmpContents() compared unequal"},
		{[2]string{"ABC", "AB"}, false, 0, "Unequal length cmpContents() compared equal"},
		{[2]string{"ABCD", ""}, false, 0, "Empty and non-empty cmpContents() compared equal"},
		{[2]string{"A", "B"}, false, 2, "Unequal length 1 cmpContents() compared equal"},

		{[2]string{makeString("X", minCmpBufSize), makeString("X", minCmpBufSize)},
			true, 2 * minCmpBufSize,
			"Equal length-4096 and content compared unequal"},
		{[2]string{makeString("Y", minCmpBufSize), makeString("X", minCmpBufSize)},
			false, 2 * minCmpBufSize,
			"Equal length-4096 diff content compared equal"},

		{[2]string{makeString("X", minCmpBufSize+1), makeString("X", minCmpBufSize+1)},
			true, 2 * (minCmpBufSize + 1),
			"Equal length-4097 and content compared unequal"},
		{[2]string{makeString("Y", minCmpBufSize+1), makeString("X", minCmpBufSize+1)},
			false, 2 * minCmpBufSize,
			"Equal length-4097 diff contents compared equal"},

		{[2]string{makeString("X", minCmpBufSize), makeString("X", minCmpBufSize+1)},
			false, 2 * minCmpBufSize,
			"Unequal lengths cmpContents() compared equal"},

		{[2]string{makeString("X", 2*minCmpBufSize), makeString("X", 2*minCmpBufSize)},
			true, 4 * minCmpBufSize,
			"Equal lengths cmpContents() compared unequal"},
	}

	for _, v := range tests {
		// Initialize buffers with different content, to test that the funcs
		// don't use unread bytes in their comparisons (particularly when not
		// Read returns less than the full slice len)
		initDifferentBufs(t, s.cmpBuf1, s.cmpBuf2)

		s.Results.BytesCompared = 0 // Reset BytesCompared
		simpleFileMaker(t, pathContents{"f1": v.content[0], "f2": v.content[1]})
		got, err := areFileContentsEqual(s, "f1", "f2")
		if v.wants != got || err != nil {
			t.Errorf(v.errStr)
		}
		if v.bytesCompared != s.Results.BytesCompared {
			t.Errorf("Incorrect BytesCompared. Expected %v, got %v", v.bytesCompared, s.Results.BytesCompared)
		}
		os.Remove("f1")
		os.Remove("f2")
	}
}
