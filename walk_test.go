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
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

type testIncludesExcludes struct {
	in    []string
	ex    []string
	match string
	n     int
}

func TestWalkFileIncludes(t *testing.T) {
	topdir, err := ioutil.TempDir("", "hardlinkable")
	if err != nil {
		t.Fatalf("Couldn't create temp dir for dir excludes tests: %v", err)
	}
	defer os.RemoveAll(topdir)

	if os.Chdir(topdir) != nil {
		t.Fatalf("Couldn't chdir to temp dir for dir excludes tests")
	}

	filenameMap := map[string]struct{}{
		"f1":       struct{}{},
		"*f2":      struct{}{},
		"f3.*.txt": struct{}{},
		"f4.*.raw": struct{}{},
		"f5":       struct{}{},
	}
	for pattern, _ := range filenameMap {
		f, err := ioutil.TempFile(topdir, pattern)
		if err != nil {
			t.Fatalf("Couldn't create tempfile with pattern: '%v' in dir: '%v'", pattern, topdir)
		}
		defer os.Remove(f.Name())
	}
	dirs := []string{topdir}

	incExSlice := []testIncludesExcludes{
		testIncludesExcludes{[]string{}, []string{}, "", len(filenameMap)},
		testIncludesExcludes{[]string{"f1"}, []string{}, "f1", 1},
		testIncludesExcludes{[]string{"f1"}, []string{"f1"}, "", len(filenameMap)},
		testIncludesExcludes{[]string{"f1"}, []string{"f1", "f2"}, "", len(filenameMap) - 1},
		testIncludesExcludes{[]string{"f1", "f2"}, []string{}, "f1", 2},
		testIncludesExcludes{[]string{"f1", "f2"}, []string{}, "f2", 2},
		testIncludesExcludes{[]string{"f1", "f2", "f3", "f4", "f5"}, []string{}, "f5", 5},
		testIncludesExcludes{[]string{"f1", "f2", "f3", "f4", "f5", "f6"}, []string{}, "f5", 5},
	}

	options := Options{}
	stats := newLinkingStats(&options)
	for _, v := range incExSlice {
		options.FileIncludes = v.in
		options.FileExcludes = v.ex

		c := stats.MatchedPathnames(dirs, []string{}, options)
		n := 0
		var filenames []string
		foundMatch := false
		for pathname := range c {
			n++
			_, filename := path.Split(pathname)
			filenames = append(filenames, filename)
			if strings.Contains(filename, v.match) {
				foundMatch = true
			}
		}
		if !foundMatch {
			t.Errorf("Included pattern '%v' not found in filenames: %v", v.match, filenames)
		}
		if n != v.n {
			t.Errorf("Expected %v files for '%v' included match, got: %v", v.n, v.match, n)
		}
	}
}
