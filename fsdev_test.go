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
	"testing"
)

func TestLinkedInoSet(t *testing.T) {
	options := &Options{}
	results := newResults(options)
	ws := status{options, results, nil}
	fs := newFSDev(ws, 10000, 10000) // Arbitrary args

	// Test when no linkable inos have been added yet
	s := fs.linkedInoSet(1)
	if len(s) != 1 && !s.Has(1) {
		t.Errorf("Linked InoSet was expected to contain just {1}: %v", s)
	}

	// Create a list of cumulative linkable ino tests
	var tests = []struct {
		pairs [2]uint64
		get   uint64
		has   []uint64
	}{
		// A pair of linked inos
		{[2]uint64{1, 2}, 1, []uint64{1, 2}},
		{[2]uint64{2, 1}, 2, []uint64{1, 2}},

		// Another group of linked inos
		{[2]uint64{3, 4}, 3, []uint64{4, 3}},
		{[2]uint64{4, 3}, 4, []uint64{3, 4}},
		{[2]uint64{3, 5}, 5, []uint64{3, 4, 5}},
		{[2]uint64{4, 5}, 4, []uint64{3, 4, 5}},
		{[2]uint64{5, 4}, 3, []uint64{3, 4, 5}},
		{[2]uint64{5, 3}, 3, []uint64{3, 4, 5}},

		// Link the two separate groups
		{[2]uint64{2, 3}, 1, []uint64{1, 2, 3, 4, 5}},

		// Make 3 overlapping pairs, then link all groups
		{[2]uint64{6, 7}, 6, []uint64{6, 7}},
		{[2]uint64{7, 8}, 7, []uint64{6, 7, 8}},
		{[2]uint64{8, 9}, 8, []uint64{6, 7, 8, 9}},
		{[2]uint64{5, 6}, 6, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}

	for _, v := range tests {
		fs.addLinkableInos(v.pairs[0], v.pairs[1])
		s = fs.linkedInoSet(v.get)
		if len(s) != len(v.has) {
			t.Errorf("Expected InoSet len to be : %v, got: %v", len(v.has), len(s))
		}
		if !s.HasAll(v.has...) {
			t.Errorf("Expected InoSet to be : %v, got: %v", v.has, s.AsSlice())
		}
	}
}
