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

func TestPathsplitSet(t *testing.T) {
	var s pathsplitSet
	s = newPathsplitSet()
	if len(s) != 0 {
		t.Errorf("Empty pathsplitSet length isn't 0: %v", s)
	}
	s = newPathsplitSet(SplitPathname("/a/a"), SplitPathname("/a/b"), SplitPathname("/b/a"))
	if len(s) != 3 {
		t.Errorf("Length %d pathsplitSet should be 3", len(s))
	}
	s.add(SplitPathname("/c/b"))
	if len(s) != 4 {
		t.Errorf("Length %d pathsplitSet should be 4", len(s))
	}
	s.remove(SplitPathname("/a/a"))
	if len(s) != 3 {
		t.Errorf("Length %d pathsplitSet should be 3", len(s))
	}
	c := s.clone()
	if !reflect.DeepEqual(s, c) {
		t.Errorf("pathsplitSet clone: %v is unequal to original: %v", c, s)
	}
	c.add(SplitPathname("/c/b"))
	if !reflect.DeepEqual(s, c) {
		t.Errorf("pathsplitSet clone: %v is unequal to original: %v", c, s)
	}
	c.remove(SplitPathname("/c/b"))
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
	f.add(SplitPathname("/a/a"))
	if len(f.pMap) != 1 {
		t.Errorf("Length %d of filenamePaths pMap should be 1", len(f.pMap))
	}
	if f.isEmpty() {
		t.Errorf("isEmpty() on non-empty filenamePaths is wrong")
	}
	f.add(SplitPathname("/b/a"))
	if len(f.pMap) != 1 {
		t.Errorf("Length %d of filenamePaths pMap should be 1", len(f.pMap))
	}

	// Add a new path with a unique filename
	f.add(SplitPathname("/a/c"))
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
	f.remove(SplitPathname("/a/a"))
	if len(f.pMap) != 2 {
		t.Errorf("Length %d of filenamePaths pMap should be 2", len(f.pMap))
	}
	if len(f.pMap["a"]) != 1 {
		t.Errorf("Length %d of filenamePaths pMap[\"a\"] should be 1", len(f.pMap["a"]))
	}
}
