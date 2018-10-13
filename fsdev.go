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
	Dev          uint64
	MaxNLinks    uint64
	inoHashes    I.InoHashes
	inoStatInfo  I.InoStatInfo
	InoPaths     I.PathsMap
	LinkableInos I.LinkableInoSets
	I.InoDigests
	pool *P.StringPool
}

func newFSDev(lstatus status, dev, maxNLinks uint64) fsDev {
	var w = fsDev{
		status:       lstatus,
		Dev:          dev,
		MaxNLinks:    maxNLinks,
		inoHashes:    make(I.InoHashes),
		inoStatInfo:  make(I.InoStatInfo),
		InoPaths:     make(I.PathsMap),
		LinkableInos: make(I.LinkableInoSets),
		InoDigests:   I.NewInoDigests(),
		pool:         P.NewPool(),
	}

	return w
}

// For a given pathname, determine which inode it is linked to, and how that
// inode relates to other walked inodes (ie. what are the existing inode links,
// and whether the inode and file contents allow it to be linked to another
// inode).
//
// This function is mainly concerned with how the inodes (and their contents)
// relate to each other.  Determining which pathnames to move from inode to
// inode (including those with the "same name" restriction), is done at a later
// stage, after all the inode relationships are discovered.
//
// An error is returned if there was a problem reading files during a
// comparison.
func (f *fsDev) FindIdenticalFiles(di I.DevStatInfo, pathname string) (err error) {
	panicIf(f.Dev != di.Dev, "Mismatched Dev %d for %s\n", f.Dev, pathname)
	curPath := P.Split(pathname, f.pool)
	curPS := I.PathInfo{Pathsplit: curPath, StatInfo: di.StatInfo}
	ino := di.StatInfo.Ino

	if _, ok := f.inoStatInfo[ino]; !ok {
		f.Results.foundInode(di.StatInfo.Nlink)
	}

	// Compute a "hash" from inode stat info, and store it if new.  If it's
	// a previously seen inode hash, check to see if one of the previously
	// seen inodes with that hash also has identical file contents.
	o := f.Options
	H := I.HashIno(di.StatInfo, o.IgnoreTime, o.IgnorePerm, o.IgnoreOwner)
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
		li := f.LinkableInos.Containing(ino)
		hi := f.inoHashes[H]
		linkableHashedInos := li.Intersection(hi)
		foundLinkableHashedInos := len(linkableHashedInos) > 0
		if !foundLinkableHashedInos {
			// Get a list of previously seen inodes that may be linkable
			cachedSeq, useDigest := f.cachedInos(H, curPS)

			// Search the list of potential inodes, looking for a match
			f.Results.searchedInoSeq()
			foundLinkable := false
			for _, cachedIno := range cachedSeq {
				f.Results.incInoSeqIterations()
				cachedPS := f.PathInfoFromIno(cachedIno)

				var areLinkable bool
				areLinkable, err = f.areFilesLinkable(cachedPS, curPS, useDigest)
				if areLinkable {
					f.LinkableInos.Add(cachedPS.Ino, ino)
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

	return
}

// cachedInos returns a slice of inos that can be searched for equal contents.
// Also return true if searching by file content digests was enabled (triggered
// by the length of the search list for the given hash exceeding a threshold).
func (f *fsDev) cachedInos(H I.Hash, ps I.PathInfo) ([]I.Ino, bool) {
	var cachedSeq []I.Ino
	cachedSet := f.inoHashes[H]
	// If digests are enabled, and cached inode lists are
	// long enough, then switch on the use of digests.
	thresh := f.Options.SearchThresh
	useDigest := thresh >= 0 && len(cachedSet) > thresh
	if useDigest {
		digest, err := I.ContentDigest(ps.Pathsplit.Join(), f.digestBuf)
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

// Return a PathInfo for the given Ino, chosen from our stored path/stat data
func (f *fsDev) PathInfoFromIno(ino I.Ino) I.PathInfo {
	path := f.InoPaths.ArbitraryPath(ino)
	fi := f.inoStatInfo[ino]
	return I.PathInfo{Pathsplit: path, StatInfo: *fi}
}

// Return true if the files have compatible inode params and equal file
// content.  Return error if file io errors occurred.
func (f *fsDev) areFilesLinkable(pi1 I.PathInfo, pi2 I.PathInfo, useDigest bool) (bool, error) {
	// Dev is equal for both PathInfos
	if pi1.Ino == pi2.Ino {
		return false, nil
	}
	if pi1.Size != pi2.Size {
		return false, nil
	}
	if !f.Options.IgnoreTime && !pi1.EqualTime(pi2) {
		return false, nil
	}
	if !f.Options.IgnorePerm && !pi1.EqualMode(pi2) {
		return false, nil
	}
	if !f.Options.IgnoreOwner && !pi1.EqualOwnership(pi2) {
		return false, nil
	}
	if !f.Options.IgnoreXattr {
		if eq, _ := I.EqualXAttrs(pi1.Join(), pi2.Join()); !eq {
			return false, nil
		}
	}

	if useDigest {
		if f.InoDigests.NewDigest(pi1, f.digestBuf) {
			f.Results.computedDigest()
		}
		if f.InoDigests.NewDigest(pi2, f.digestBuf) {
			f.Results.computedDigest()
		}
	}

	f.Results.didComparison()
	eq, err := areFileContentsEqual(f.status, pi1.Join(), pi2.Join())
	if err != nil {
		return false, err
	}

	// If two equal files are found, determine if any of the ignored inode
	// parameters would have precluded returning a true value, had they not
	// been ignored (and record in the Results).
	if eq {
		f.Results.foundEqualFiles()

		// Add some debugging statistics for files that are found to be
		// equal, but which have some mismatched inode parameters.
		addMismatchTotalBytes := false
		if !pi1.EqualTime(pi2) {
			f.Results.addMismatchedMtimeBytes(pi1.Size)
			addMismatchTotalBytes = true
		}
		if !pi1.EqualMode(pi2) {
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
	return eq, nil
}
