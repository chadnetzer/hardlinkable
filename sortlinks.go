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
	"log"
	"sort"

	I "github.com/chadnetzer/hardlinkable/internal/inode"
	P "github.com/chadnetzer/hardlinkable/internal/pathpool"
)

// Implement sorting from greatest NLink count to least
type inoNlink struct {
	Ino   I.Ino
	Nlink uint64
}
type byNlink []inoNlink

func (a byNlink) Len() int { return len(a) }
func (a byNlink) Less(i, j int) bool {
	return a[i].Nlink < a[j].Nlink || (a[i].Nlink == a[j].Nlink && a[i].Ino > a[j].Ino)
}
func (a byNlink) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (f *fsDev) sortSetByNlink(inoSet I.Set) []I.Ino {
	seq := make(byNlink, len(inoSet))
	i := 0
	for ino := range inoSet {
		nlink := f.inoStatInfo[ino].Nlink
		seq[i] = inoNlink{Ino: ino, Nlink: nlink}
		i++
	}

	sort.Sort(sort.Reverse(seq))

	sortedSeq := make([]I.Ino, len(seq))
	for i, inoNlink := range seq {
		sortedSeq[i] = inoNlink.Ino
	}

	return sortedSeq
}

// Reverse fromS and append to toS
func appendReversedInos(toS []I.Ino, fromS ...I.Ino) []I.Ino {
	for i, j := 0, len(fromS)-1; i < j; i, j = i+1, j-1 {
		fromS[i], fromS[j] = fromS[j], fromS[i]
	}
	return append(toS, fromS...)
}

// generateLinks takes the sets of matching inodes that were discovered by the
// fsDev.FindIdenticalFiles() method, and generates lists of hardlinkable
// pathnames (with a src pathname to be linked to multiple destination
// pathnames) from them, optionally performing the actual linking as well.
//
// For every discovered set of matching inodes, this method sorts the inodes by
// nlink count from highest to lowest, and generally proceeds by linking the
// inodes with the highest nlink count to those with the lowest nlink count
// (until the maximum nlink count is reached, at which point it proceeds to the
// src inode with the next highest nlink count).
func (f *fsDev) generateLinks() error {
	for linkableSet := range f.LinkableInos.All() {
		// Sort links highest nlink to lowest
		sortedInos := f.sortSetByNlink(linkableSet)
		if err := f.genLinksHelper(sortedInos); err != nil {
			return err
		}
	}
	return nil
}

// genLinksHelper operates on the set of matching inodes, sorted from highest
// nlink count to lowest.  It selects the set of src and dst pathnames that
// will (ideally) link all the inodes together.  It respects the maximum nlink
// count, which can prevent all the inodes from being consolidated into one.
// It also respects the "same name" option, which only generates linked
// pathnames that have equal (case-sensitive) filenames, which can also prevent
// all the inodes in the set from being consolidated.
func (f *fsDev) genLinksHelper(sortedInos []I.Ino) error {
	remainingInos := make([]I.Ino, 0)

	// The remainingInos are the inodes at the far end of the sorted inode
	// list, which were skipped over on a previous linking pass because
	// of a restriction such as the optional "same name" linking
	// requirement.
	for len(sortedInos) > 0 || len(remainingInos) > 0 {
		if len(remainingInos) > 0 {
			// Reverse remainingInos and place at the end
			// of sortedInos.  These were the leftovers
			// from the end of the list working backwards.
			sortedInos = appendReversedInos(sortedInos, remainingInos...)
			remainingInos = make([]I.Ino, 0)
		}
		// Advance the srcIno forward by one inode, and loop over
		// the remaining inodes looking for a potential link.
		srcIno := sortedInos[0]
		sortedInos = sortedInos[1:]
		for len(sortedInos) > 0 {
			dstIno := sortedInos[len(sortedInos)-1]
			sortedInos = sortedInos[:len(sortedInos)-1]
			srcSI := f.inoStatInfo[srcIno]
			dstSI := f.inoStatInfo[dstIno]

			// Check if max NLinks would be exceeded if
			// these two inodes are fully linked
			sum := uint64(srcSI.Nlink) + uint64(dstSI.Nlink)
			if sum > f.MaxNLinks {
				remainingInos = append(remainingInos, dstIno)
				remainingInos = appendReversedInos(remainingInos, sortedInos...)
				sortedInos = make([]I.Ino, 0)
				break
			}

			// For a given dst inode, iterate over all the paths
			// linking to it, using the pathnames for linking,
			// while respecting both the SameName option and the
			// maximum src inode nlink count.
			dstPaths := f.InoPaths.AllPaths(dstIno)
			for dstPath := range dstPaths {
				var srcPath P.Pathsplit
				if f.Options.SameName {
					// Skip to next destination inode path if dst filename
					// isn't also found as a src filename
					srcPaths := f.InoPaths[srcIno]
					dstFilename := dstPath.Filename
					if !srcPaths.HasFilename(dstFilename) {
						continue
					}
					srcPath = f.InoPaths.ArbitraryFilenamePath(srcIno, dstFilename)
				} else {
					srcPath = f.InoPaths.ArbitraryPath(srcIno)
				}
				srcPathInfo := I.PathInfo{Pathsplit: srcPath, StatInfo: *srcSI}
				dstPathInfo := I.PathInfo{Pathsplit: dstPath, StatInfo: *dstSI}

				// Abort if the filesystem is found to be "active" (ie. changing)
				if f.Options.CheckQuiescence || f.Options.LinkingEnabled {
					modifiedErr := f.haveNotBeenModified(srcPathInfo, dstPathInfo)
					if modifiedErr != nil {
						return modifiedErr
					}
				}

				// Perform the actual linking if requested, but abort all remaining
				// linking if a linking error is encountered.
				var linkingErr error
				if f.Options.LinkingEnabled {
					linkingErr = f.hardlinkFiles(srcPathInfo, dstPathInfo)
					if linkingErr != nil {
						if !f.Options.IgnoreLinkErrors {
							return linkingErr
						} else if f.Options.DebugLevel > 0 {
							log.Printf("\r%v  Skipping...", linkingErr)
						}
					}
				}

				if linkingErr != nil {
					f.Results.skippedNewLink(srcPath, dstPath)
				} else {
					f.Results.foundNewLink(srcPath, dstPath)

					// Update cached StatInfo information for inodes
					srcSI.Nlink++
					dstSI.Nlink--
					if dstSI.Nlink == 0 {
						f.Results.foundRemovedInode(dstSI.Size)
						delete(f.inoStatInfo, dstIno)
					}
					f.InoPaths.MovePath(dstPath, srcIno, dstIno)
				}
			}
			// With SameName option, it's possible that the dstIno nLinks will not go
			// to zero (if not all links have a matching filename), so place on the
			// remainingInos list to allow it to (possibly) be linked with other inodes
			fp, ok := f.InoPaths[dstIno]
			if ok && !fp.IsEmpty() {
				remainingInos = append(remainingInos, dstIno)
			}
		}
	}
	return nil
}
