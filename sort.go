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
	"sort"
)

// Implement sorting from greatest NLink count to least
type inoNlink struct {
	Ino   uint64
	Nlink uint32
}
type byNlink []inoNlink

func (a byNlink) Len() int { return len(a) }
func (a byNlink) Less(i, j int) bool {
	return a[i].Nlink < a[j].Nlink || (a[i].Nlink == a[j].Nlink && a[i].Ino > a[j].Ino)
}
func (a byNlink) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (f *FSDev) sortInoSetByNlink(inoSet InoSet) []Ino {
	seq := make(byNlink, len(inoSet))
	i := 0
	for ino, _ := range inoSet {
		nlink := f.InoStatInfo[ino].Nlink
		seq[i] = inoNlink{Ino: ino, Nlink: nlink}
		i++
	}

	sort.Sort(sort.Reverse(seq))

	sortedSeq := make([]Ino, len(seq))
	for i, inoNlink := range seq {
		sortedSeq[i] = inoNlink.Ino
	}

	return sortedSeq
}

// Reverse fromS and append to toS
func appendReversedInos(toS []Ino, fromS ...Ino) []Ino {
	for i, j := 0, len(fromS)-1; i < j; i, j = i+1, j-1 {
		fromS[i], fromS[j] = fromS[j], fromS[i]
	}
	return append(toS, fromS...)
}

func (f *FSDev) sortedLinks() <-chan PathStatPair {
	out := make(chan PathStatPair)
	go func() {
		defer close(out)
		c := f.linkedInoSets()
		for linkableSet := range c {
			// Sort links highest nlink to lowest
			sortedInos := f.sortInoSetByNlink(linkableSet)
			f.sendLinkedPairs(sortedInos, out)
		}
	}()
	return out
}

func (f *FSDev) sendLinkedPairs(sortedInos []Ino, out chan<- PathStatPair) {
	remainingInos := make([]Ino, 0)

	for len(sortedInos) > 0 || len(remainingInos) > 0 {
		if len(remainingInos) > 0 {
			// Reverse remainingInos and place at the end
			// of sortedInos.  These were the leftovers
			// from the end of the list working backwards.
			sortedInos = appendReversedInos(sortedInos, remainingInos...)
			remainingInos = make([]Ino, 0)
		}
		srcIno := sortedInos[0]
		sortedInos = sortedInos[1:]
		for len(sortedInos) > 0 {
			dstIno := sortedInos[len(sortedInos)-1]
			sortedInos = sortedInos[:len(sortedInos)-1]
			srcSI := f.InoStatInfo[srcIno]
			dstSI := f.InoStatInfo[dstIno]

			// Check if max NLinks would be exceeded if
			// these two inodes are fully linked
			sum := uint64(srcSI.Nlink) + uint64(dstSI.Nlink)
			if sum > f.MaxNLinks {
				remainingInos = append(remainingInos, dstIno)
				remainingInos = appendReversedInos(remainingInos, sortedInos...)
				sortedInos = make([]Ino, 0)
				break
			}

			dstPaths := f.allInoPaths(dstIno)
			for dstPath := range dstPaths {
				var srcPath Pathsplit
				if MyOptions.SameName {
					// Skip to next destination inode path if dst filename
					// isn't also found as a src filename
					srcPaths := f.InoPaths[srcIno].pMap
					dstFilename := dstPath.Filename
					if _, ok := srcPaths[dstFilename]; !ok {
						continue
					}
					srcPath = f.ArbitraryFilenamePath(srcIno, dstFilename)
				} else {
					srcPath = f.ArbitraryPath(srcIno)
				}
				srcPathStat := PathStat{srcPath, srcSI}
				dstPathStat := PathStat{dstPath, dstSI}

				out <- PathStatPair{srcPathStat, dstPathStat}

				Stats.FoundNewLink(srcPathStat, dstPathStat)

				// Update StatInfo information for inodes
				srcSI.Nlink += 1
				dstSI.Nlink -= 1
				f.InoStatInfo[srcIno] = srcSI
				if dstSI.Nlink == 0 {
					delete(f.InoStatInfo, dstIno)
				} else {
					f.InoStatInfo[dstIno] = dstSI
				}
				f.moveLinkedPath(dstPath, srcIno, dstIno)
			}
			// With SameName option, it's possible that the dstIno nLinks will not go
			// to zero (if not all links have a matching filename), so place on the
			// remainingInos list to allow it to (possibly) be linked with other linked
			// inodes
			fp, ok := f.InoPaths[dstIno]
			if ok && !fp.isEmpty() {
				remainingInos = append(remainingInos, dstIno)
			}
		}
	}
}
