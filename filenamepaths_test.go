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
	P "hardlinkable/internal/pathpool"
	"reflect"
	"testing"
)

func SP(pathname string) P.Pathsplit {
	return P.Split(pathname, nil)
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
	if len(f.PMap) != 0 {
		t.Errorf("Empty FilenamePaths length isn't 0: %v", f)
	}
	if !f.IsEmpty() {
		t.Errorf("isEmpty() on empty FilenamePaths is wrong")
	}

	// Add two separate paths with same filename (ie. basename)
	f.Add(SP("/a/a"))
	if len(f.PMap) != 1 {
		t.Errorf("Length %d of FilenamePaths PMap should be 1", len(f.PMap))
	}
	if f.IsEmpty() {
		t.Errorf("isEmpty() on non-empty FilenamePaths is wrong")
	}
	f.Add(SP("/b/a"))
	if len(f.PMap) != 1 {
		t.Errorf("Length %d of FilenamePaths PMap should be 1", len(f.PMap))
	}

	// Add a new path with a unique filename
	f.Add(SP("/a/c"))
	if len(f.PMap) != 2 {
		t.Errorf("Length %d of FilenamePaths PMap should be 2", len(f.PMap))
	}
	if len(f.PMap["a"]) != 2 {
		t.Errorf("Length %d of FilenamePaths PMap[\"a\"] should be 2", len(f.PMap["a"]))
	}
	if len(f.PMap["c"]) != 1 {
		t.Errorf("Length %d of FilenamePaths PMap[\"c\"] should be 1", len(f.PMap["c"]))
	}

	// Remove one of the path's with "a" filename
	f.Remove(SP("/a/a"))
	if len(f.PMap) != 2 {
		t.Errorf("Length %d of FilenamePaths PMap should be 2", len(f.PMap))
	}
	if len(f.PMap["a"]) != 1 {
		t.Errorf("Length %d of FilenamePaths PMap[\"a\"] should be 1", len(f.PMap["a"]))
	}

	c := f.Copy()
	if !reflect.DeepEqual(f, c) {
		t.Errorf("FilenamePaths clone: %v is unequal to original: %v", c, f)
	}

	c.Add(SP("/c/b"))
	if reflect.DeepEqual(f, c) {
		t.Errorf("FilenamePaths clone: %v is equal to original: %v after added path", c, f)
	}

	z := c.AnyWithFilename("c")
	if z.Filename != "c" {
		t.Errorf("FilenamePaths anyWithFilename(\"c\") returned wrong filename: %v", z)
	}

	x := c.Any()
	y := c.Any()
	if x != y {
		t.Errorf("FilenamePaths any() returned two different values: %v %v", x, y)
	}
	c.Remove(x)
	x = c.Any()
	if x == y {
		t.Errorf("FilenamePaths any() returned removed path: %v", y)
	}
}
