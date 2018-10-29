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

// +build slowtests

package hardlinkable

import (
	"fmt"
	"os"
	"sort"
	"syscall"
	"testing"

	"github.com/chadnetzer/hardlinkable/internal/inode"
)

func inoVal(pathname string) uint64 {
	l, err := os.Lstat(pathname)
	if err != nil {
		return 0
	}
	statT, ok := l.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint64(statT.Ino)
}

func TestMaxNlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MaxNlink test in short mode")
	} else {
		t.Log("Use -short option to skip MaxNlinks test")
	}
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	m := pathContents{"f1": "X"}
	simpleFileMaker(t, m)

	N := inode.MaxNlinkVal("f1")
	if N > (1<<15 - 1) {
		t.Skip("Skipping MaxNlink test because Nlink max is greater than 32767")
	}

	opts := SetupOptions(LinkingEnabled)

	m = pathContents{}
	for i := 0; i < int(N+100); i++ {
		filename := fmt.Sprintf("n%v", i)
		m[filename] = "Y"
	}
	simpleFileMaker(t, m)

	name := "testname: 'MaxNlinks'"
	simpleRun(name, t, opts, 2, ".")
	verifyContents(name, t, m)

	counts := make(map[int]int)
	for i := 0; i < int(2*N+100); i++ {
		filename := fmt.Sprintf("n%v", i)
		n := int(inoVal(filename))
		counts[n]++
	}

	foundNlinks := []int{}
	for _, v := range counts {
		foundNlinks = append(foundNlinks, v)
	}
	sort.Ints(foundNlinks)
	expectedNlinks := []int{100, int(N), int(N)}

	for i, v := range foundNlinks {
		if v != expectedNlinks[i] {
			t.Errorf("Nlink and leftover counts, expected: %v, got: %v", expectedNlinks, foundNlinks)
			break
		}
	}
}
