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
