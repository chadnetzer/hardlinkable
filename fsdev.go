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
	I "hardlinkable/internal/inode"
	P "hardlinkable/internal/pathpool"
)

type fsDev struct {
	status
	Dev         uint64
	MaxNLinks   uint64
	inoHashes   I.InoHashes
	inoStatInfo I.InoStatInfo
	InoPaths    I.PathsMap
	LinkedInos  I.LinkedInoSets
	I.InoDigests
	pool P.StringPool
}

func newFSDev(lstatus status, dev, maxNLinks uint64) fsDev {
	var w = fsDev{
		status:      lstatus,
		Dev:         dev,
		MaxNLinks:   maxNLinks,
		inoHashes:   make(I.InoHashes),
		inoStatInfo: make(I.InoStatInfo),
		InoPaths:    make(I.PathsMap),
		LinkedInos:  make(I.LinkedInoSets),
		InoDigests:  I.NewInoDigests(),
		pool:        P.NewPool(),
	}

	return w
}

func (f *fsDev) FindIdenticalFiles(di I.DevStatInfo, pathname string) {
	panicIf(f.Dev != di.Dev, "Mismatched Dev %d for %s\n", f.Dev, pathname)
	curPath := P.Split(pathname, f.pool)
	curPathStat := I.PathInfo{Pathsplit: curPath, StatInfo: di.StatInfo}
	ino := di.StatInfo.Ino

	if _, ok := f.inoStatInfo[ino]; !ok {
		f.Results.foundInode(di.StatInfo.Nlink)
	}

	H := I.HashIno(di.StatInfo, f.Options.IgnoreTime)
	if _, ok := f.inoHashes[H]; !ok {
		// Setup for a newly seen hash value
		f.Results.missedHash()
		f.inoHashes[H] = I.NewSet(ino)
	} else {
		f.Results.foundHash()
		// See if the new file is an inode we've seen before
		if _, ok := f.inoStatInfo[ino]; ok {
			// If it's a path we've seen before, ignore it
			if f.InoPaths.HasPath(ino, curPath) {
				return
			}
			seenPath := f.InoPaths.ArbitraryPath(ino)
			seenSize := f.inoStatInfo[ino].Size
			f.Results.foundExistingLink(seenPath, curPath, seenSize)
		}
		// See if this inode is already one we've determined can be
		// linked to another one, in which case we can avoid repeating
		// the work of linking it again.
		li := f.LinkedInos.Containing(ino)
		hi := f.inoHashes[H]
		linkedHashedInos := li.Intersection(hi)
		foundLinkedHashedInos := len(linkedHashedInos) > 0
		if !foundLinkedHashedInos {
			// Get a list of previously seen inodes that may be linkable
			cachedSeq, useDigest := f.cachedInos(H, curPathStat)

			// Search the list of potential inodes, looking for a match
			f.Results.searchedInoSeq()
			foundLinkable := false
			for _, cachedIno := range cachedSeq {
				f.Results.incInoSeqIterations()
				cachedPathStat := f.PathInfoFromIno(cachedIno)
				if f.areFilesLinkable(cachedPathStat, curPathStat, useDigest) {
					f.LinkedInos.Add(cachedPathStat.Ino, ino)
					foundLinkable = true
					break
				}
			}
			// Add hash to set if no match was found in current set
			if !foundLinkable {
				f.Results.noHashMatch()
				inoSet := f.inoHashes[H]
				inoSet.Add(ino)
			}
		}
	}
	// Remember Inode and filename/path information for each seen file
	f.inoStatInfo[ino] = &di.StatInfo
	f.InoPaths.AppendPath(ino, curPath)
}

// possibleInos returns a slice of inos that can be searched for equal contents
func (f *fsDev) cachedInos(H I.Hash, ps I.PathInfo) ([]I.Ino, bool) {
	var cachedSeq []I.Ino
	cachedSet := f.inoHashes[H]
	// If digests are enabled, and cached inode lists are
	// long enough, then switch on the use of digests.
	thresh := f.Options.SearchThresh
	useDigest := thresh >= 0 && len(cachedSet) > thresh
	if useDigest {
		digest, err := I.ContentDigest(ps.Pathsplit.Join())
		if err == nil {
			// With digests, we take the (potentially long) set of cached inodes (ie.
			// those inodes that all have the same InoHash), and remove the inodes that
			// are definitely not a match because their digests do not match with the
			// current inode.  We also put the inodes with equal digests before those
			// that have no digest yet, in hopes of more quickly finding an identical file.
			f.Results.computedDigest()
			f.InoDigests.Add(ps, digest)
			noDigests := cachedSet.Difference(f.InosWithDigest)
			sameDigests := cachedSet.Intersection(f.InoDigests.GetInos(digest))
			differentDigests := cachedSet.Difference(sameDigests).Difference(noDigests)
			cachedSeq = append(sameDigests.AsSlice(), noDigests.AsSlice()...)

			panicIf(noDigests.Has(ps.StatInfo.Ino), "New Ino found in noDigests\n")
			panicIf(len(I.SetIntersections(sameDigests, differentDigests, noDigests)) > 0,
				"Overlapping digest sets\n")
		}
	} else {
		cachedSeq = cachedSet.AsSlice()
	}

	return cachedSeq, useDigest
}

func (f *fsDev) PathInfoFromIno(ino I.Ino) I.PathInfo {
	path := f.InoPaths.ArbitraryPath(ino)
	fi := f.inoStatInfo[ino]
	return I.PathInfo{Pathsplit: path, StatInfo: *fi}
}

func (f *fsDev) areFilesLinkable(pi1 I.PathInfo, pi2 I.PathInfo, useDigest bool) bool {
	// Dev is equal for both PathInfos
	if pi1.Ino == pi2.Ino {
		return false
	}
	if pi1.Size != pi2.Size {
		return false
	}
	if !f.Options.IgnoreTime && !pi1.EqualTime(pi2) {
		return false
	}
	if !f.Options.IgnorePerms && !pi1.EqualMode(pi2) {
		return false
	}
	if !f.Options.IgnoreOwner && !pi1.EqualOwnership(pi2) {
		return false
	}
	if !f.Options.IgnoreXattr {
		if eq, _ := I.EqualXAttrs(pi1.Join(), pi2.Join()); !eq {
			return false
		}
	}

	if useDigest {
		if wasComputed := f.InoDigests.NewDigest(pi1); wasComputed {
			f.Results.computedDigest()
		}
		if wasComputed := f.InoDigests.NewDigest(pi2); wasComputed {
			f.Results.computedDigest()
		}
	}

	f.Results.didComparison()
	eq, _ := areFileContentsEqual(f.status, pi1.Join(), pi2.Join())
	if eq {
		f.Results.foundEqualFiles()

		// Add some debugging statistics for files that are found to be
		// equal, but which have some mismatched inode parameters.
		addMismatchTotalBytes := false
		if !(pi1.EqualTime(pi2)) {
			f.Results.addMismatchedMtimeBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		if pi1.EqualMode(pi2) {
			f.Results.addMismatchedModeBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		if pi1.Uid != pi2.Uid {
			f.Results.addMismatchedUidBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		if pi1.Gid != pi2.Gid {
			f.Results.addMismatchedGidBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		eqX, err := I.EqualXAttrs(pi1.Join(), pi2.Join())
		if err == nil && !eqX {
			f.Results.addMismatchedXattrBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		if addMismatchTotalBytes {
			f.Results.addMismatchedTotalBytes(pi1.Size)
		}
	}
	return eq
}
