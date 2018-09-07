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
	"reflect"
	"testing"
)

func SP(pathname string) Pathsplit {
	return SplitPathname(pathname, nil)
}

func TestPathsplitSet(t *testing.T) {
	var s pathsplitSet
	s = newPathsplitSet()
	if len(s) != 0 {
		t.Errorf("Empty pathsplitSet length isn't 0: %v", s)
	}
	s = newPathsplitSet(SP("/a/a"), SP("/a/b"), SP("/b/a"))
	if len(s) != 3 {
		t.Errorf("Length %d pathsplitSet should be 3", len(s))
	}
	s.add(SP("/c/b"))
	if len(s) != 4 {
		t.Errorf("Length %d pathsplitSet should be 4", len(s))
	}
	s.remove(SP("/a/a"))
	if len(s) != 3 {
		t.Errorf("Length %d pathsplitSet should be 3", len(s))
	}
	c := s.clone()
	if !reflect.DeepEqual(s, c) {
		t.Errorf("pathsplitSet clone: %v is unequal to original: %v", c, s)
	}
	c.add(SP("/c/b")) // Adding path a second time
	if !reflect.DeepEqual(s, c) {
		t.Errorf("pathsplitSet clone: %v is unequal to original: %v", c, s)
	}
	c.remove(SP("/c/b"))
	if reflect.DeepEqual(s, c) {
		t.Errorf("After path removal pathsplitSet clone: %v is equal to original: %v", c, s)
	}
	c = s.clone()
	p := s.any()
	c.remove(p)
	c.add(p)
	if !reflect.DeepEqual(s, c) {
		t.Errorf("pathsplitSet clone: %v is unequal to original: %v", c, s)
	}
}

func TestFilenamePaths(t *testing.T) {
	var f *filenamePaths
	f = newFilenamePaths()
	if len(f.pMap) != 0 {
		t.Errorf("Empty filenamePaths length isn't 0: %v", f)
	}
	if !f.isEmpty() {
		t.Errorf("isEmpty() on empty filenamePaths is wrong")
	}

	// Add two separate paths with same filename (ie. basename)
	f.add(SP("/a/a"))
	if len(f.pMap) != 1 {
		t.Errorf("Length %d of filenamePaths pMap should be 1", len(f.pMap))
	}
	if f.isEmpty() {
		t.Errorf("isEmpty() on non-empty filenamePaths is wrong")
	}
	f.add(SP("/b/a"))
	if len(f.pMap) != 1 {
		t.Errorf("Length %d of filenamePaths pMap should be 1", len(f.pMap))
	}

	// Add a new path with a unique filename
	f.add(SP("/a/c"))
	if len(f.pMap) != 2 {
		t.Errorf("Length %d of filenamePaths pMap should be 2", len(f.pMap))
	}
	if len(f.pMap["a"]) != 2 {
		t.Errorf("Length %d of filenamePaths pMap[\"a\"] should be 2", len(f.pMap["a"]))
	}
	if len(f.pMap["c"]) != 1 {
		t.Errorf("Length %d of filenamePaths pMap[\"c\"] should be 1", len(f.pMap["c"]))
	}

	// Remove one of the path's with "a" filename
	f.remove(SP("/a/a"))
	if len(f.pMap) != 2 {
		t.Errorf("Length %d of filenamePaths pMap should be 2", len(f.pMap))
	}
	if len(f.pMap["a"]) != 1 {
		t.Errorf("Length %d of filenamePaths pMap[\"a\"] should be 1", len(f.pMap["a"]))
	}

	c := f.clone()
	if !reflect.DeepEqual(f, c) {
		t.Errorf("filenamePaths clone: %v is unequal to original: %v", c, f)
	}

	c.add(SP("/c/b"))
	if reflect.DeepEqual(f, c) {
		t.Errorf("filenamePaths clone: %v is equal to original: %v after added path", c, f)
	}

	z := c.anyWithFilename("c")
	if z.Filename != "c" {
		t.Errorf("filenamePaths anyWithFilename(\"c\") returned wrong filename: %v", z)
	}

	x := c.any()
	y := c.any()
	if x != y {
		t.Errorf("filenamePaths any() returned two different values: %v %v", x, y)
	}
	c.remove(x)
	x = c.any()
	if x == y {
		t.Errorf("filenamePaths any() returned removed path: %v", y)
	}
}
