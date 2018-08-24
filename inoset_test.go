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

func TestInoSetAdd(t *testing.T) {
	var s InoSet
	s = NewInoSet()
	if len(s) != 0 {
		t.Errorf("Empty InoSet length isn't 0: %v", s)
	}
	s.Add(Ino(1))
	if len(s) != 1 {
		t.Errorf("Length %d InoSet should be 1", len(s))
	}
	s.Add(Ino(2))
	if len(s) != 2 {
		t.Errorf("Length %d InoSet should be 2", len(s))
	}
	s.Add(Ino(1))
	if len(s) != 2 {
		t.Errorf("Length %d InoSet after re-adding 1 should be 2", len(s))
	}
	s.Add(Ino(2))
	if len(s) != 2 {
		t.Errorf("Length %d InoSet after re-adding 2 should be 2", len(s))
	}
	s2 := NewInoSet(1, 2)
	if !reflect.DeepEqual(s, s2) {
		t.Errorf("Found unexpected unequal InoSets: %v %v", s, s2)
	}
	s.Add(3)
	if reflect.DeepEqual(s, s2) {
		t.Errorf("Found unexpected equal InoSets: %v %v", s, s2)
	}
}

func TestInoSetDifference(t *testing.T) {
	var s1, s2 InoSet
	s1 = NewInoSet()
	s2 = NewInoSet()
	diff := s1.Difference(s2)
	if len(diff) != 0 {
		t.Errorf("Empty InoSet difference length isn't 0: %v", diff)
	}

	s1.Add(Ino(1))
	diff = s1.Difference(s2)
	if len(diff) != 1 {
		t.Errorf("InoSet difference length isn't 1: %v", diff)
	}
	if !reflect.DeepEqual(diff, NewInoSet(Ino(1))) {
		t.Errorf("InoSet difference doesn't contains only 1: %v", diff)
	}

	s2.Add(Ino(1))
	diff = s1.Difference(s2)
	if len(diff) != 0 {
		t.Errorf("InoSet difference length isn't 0: %v", diff)
	}
	if !reflect.DeepEqual(diff, NewInoSet()) {
		t.Errorf("InoSet difference isn't empty: %v. sets: %v-%v", diff, s1, s2)

func TestInoSetDifference(t *testing.T) {
	s1 := NewInoSet()
	s2 := NewInoSet()
	diff1 := s1.Difference(s2)
	diff2 := s2.Difference(s1)
	if len(diff1) != 0 || len(diff2) != 0 {
		t.Errorf("Empty InoSet difference length isn't 0: %v  %v", diff1, diff2)
	}

	s1.Add(Ino(1))
	diff1 = s1.Difference(s2)
	if !reflect.DeepEqual(diff1, NewInoSet(Ino(1))) {
		t.Errorf("InoSet difference doesn't contain only 1: %v", diff1)
	}
	diff2 = s2.Difference(s1)
	if !reflect.DeepEqual(diff2, NewInoSet()) {
		t.Errorf("InoSet difference isn't empty: %v", diff2)
	}

	s2.Add(Ino(1))
	diff1 = s1.Difference(s2)
	if !reflect.DeepEqual(diff1, NewInoSet()) {
		t.Errorf("InoSet difference isn't empty: %v", diff1)
	}
	diff2 = s2.Difference(s1)
	if !reflect.DeepEqual(diff2, NewInoSet()) {
		t.Errorf("InoSet difference isn't empty: %v", diff2)
	}

	s1.Add(Ino(2))
	diff1 = s1.Difference(s2)
	if !reflect.DeepEqual(diff1, NewInoSet(Ino(2))) {
		t.Errorf("InoSet difference doesn't contain only 2: %v", diff1)
	}
	diff2 = s2.Difference(s1)
	if !reflect.DeepEqual(diff2, NewInoSet()) {
		t.Errorf("InoSet difference isn't empty: %v", diff2)
	}
}
