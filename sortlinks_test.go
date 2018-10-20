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
	"sort"
	"testing"

	I "github.com/chadnetzer/hardlinkable/internal/inode"
)

type byIno []I.Ino

func (s byIno) Len() int           { return len(s) }
func (s byIno) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byIno) Less(i, j int) bool { return s[i] < s[j] }

func InoSeqFromSet(set I.Set) []I.Ino {
	seq := make([]I.Ino, 0)
	for ino := range set {
		seq = append(seq, ino)
	}
	return seq
}

func setupInoStatInfo(fsdev *fsDev, inoSet I.Set) {
	fsdev.inoStatInfo = make(I.InoStatInfo)
	for ino := range inoSet {
		// Using any old StatInfo is fine
		di, _ := I.LStatInfo(".")
		// Deliberately make it so that if Nlinks are sorted, Inos are
		// sorted also (for easier testing of []I.Ino result)
		di.Nlink = uint64(ino)*2 + 100
		di.Ino = I.Ino(ino)
		fsdev.inoStatInfo[I.Ino(ino)] = &di.StatInfo
	}
}

func TestInoSort(t *testing.T) {
	inoSet := I.NewSet(1, 3, 5, 4, 2, 6, 7, 1, 8, 2, 11, 9, 5)
	inoSeq := InoSeqFromSet(inoSet)
	if sort.IsSorted(sort.Reverse(byIno(inoSeq))) {
		t.Errorf("inoSeq was already sorted (should be unsorted)")
	}

	fsdev := &fsDev{}
	setupInoStatInfo(fsdev, inoSet)
	inoSetSorted := fsdev.sortSetByNlink(inoSet)
	if !sort.IsSorted(sort.Reverse(byIno(inoSetSorted))) {
		t.Errorf("Sorting of InoSet by nLink value failed")
	}
}

func TestAppendReversed(t *testing.T) {
	forward := []I.Ino{1, 2, 3}
	reversed := []I.Ino{5, 4}
	forward = appendReversedInos(forward, reversed...)
	if !sort.IsSorted(byIno(forward)) {
		t.Errorf("appendReversed failure")
	}

}
