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

type Hash uint64

type StatInfos map[string]StatInfo

type PathStat struct {
	Pathsplit
	StatInfo
}

type PathStatPair struct {
	Src PathStat
	Dst PathStat
}

type FSDev struct {
	Dev            uint64
	MaxNLinks      uint64
	InoHashes      map[Hash]InoSet
	InoStatInfo    map[Ino]StatInfo
	InoPaths       map[Ino]*filenamePaths
	LinkedInos     map[Ino]InoSet
	DigestIno      map[Digest]InoSet
	InosWithDigest InoSet
	pool           internPool

	// For each directory name, keep track of all the StatInfo structures
	DirnameStatInfos map[string]StatInfos
}

func (s1 PathStat) EqualTime(s2 PathStat) bool {
	return s1.Sec == s2.Sec && s1.Nsec == s2.Nsec
}

func (s1 PathStat) EqualMode(s2 PathStat) bool {
	return s1.Mode == s2.Mode
}

func (s1 PathStat) EqualOwnership(s2 PathStat) bool {
	return s1.Uid == s2.Uid && s1.Gid == s2.Gid
}

func (f *FSDev) LinkedInosCopy() map[Ino]InoSet {
	newLinkedInos := make(map[Ino]InoSet)
	for k, v := range f.LinkedInos {
		newLinkedInos[k] = v.Copy()
	}
	return newLinkedInos
}

func NewFSDev(dev, maxNLinks uint64) FSDev {
	var w FSDev
	w.Dev = dev
	w.MaxNLinks = maxNLinks
	w.InoHashes = make(map[Hash]InoSet)
	w.InoStatInfo = make(map[Ino]StatInfo)
	w.InoPaths = make(map[Ino]*filenamePaths)
	w.LinkedInos = make(map[Ino]InoSet)
	w.DigestIno = make(map[Digest]InoSet)
	w.InosWithDigest = NewInoSet()
	w.pool = newInternPool()

	return w
}

// Produce an equal hash for potentially equal files, based only on Inode
// metadata (size, time, etc.)
func InoHash(stat StatInfo, opt *Options) Hash {
	var value Hash
	size := Hash(stat.Size)
	// The main requirement is that files that could be equal have equal
	// hashes.  It's less important if unequal files also have the same
	// hash value, since we will still compare the actual file content
	// later.
	if opt.IgnoreTime {
		value = size
	} else {
		value = size ^ Hash(stat.Sec) ^ Hash(stat.Nsec)
	}
	return value
}

func (f *FSDev) findIdenticalFiles(devStatInfo DevStatInfo, pathname string) {
	PanicIf(f.Dev != devStatInfo.Dev, "Mismatched Dev %d for %s\n", f.Dev, pathname)
	statInfo := devStatInfo.StatInfo
	curPath := SplitPathname(pathname, f.pool)
	curPathStat := PathStat{curPath, statInfo}

	if _, ok := f.InoStatInfo[statInfo.Ino]; !ok {
		Stats.FoundInode()
	}

	H := InoHash(statInfo, MyOptions)
	if _, ok := f.InoHashes[H]; !ok {
		// Setup for a newly seen hash value
		Stats.MissedHash()
		f.InoHashes[H] = NewInoSet(statInfo.Ino)
	} else {
		Stats.FoundHash()
		// See if the new file is an inode we've seen before
		if _, ok := f.InoStatInfo[statInfo.Ino]; ok {
			// If it's a path we've seen before, ignore it
			if f.haveSeenPath(statInfo.Ino, curPath) {
				return
			}
			prevPath := f.ArbitraryPath(statInfo.Ino)
			prevStatinfo := f.InoStatInfo[statInfo.Ino]
			linkPair := LinkPair{prevPath, curPath}
			existingLinkInfo := ExistingLink{linkPair, prevStatinfo}
			Stats.FoundExistingLink(existingLinkInfo)
		}
		// See if this inode is already one we've determined can be
		// linked to another one, in which case we can avoid repeating
		// the work of linking it again.
		linkedInos := f.linkedInoSet(statInfo.Ino)
		hashedInos := f.InoHashes[H]
		linkedHashedInos := linkedInos.Intersection(hashedInos)
		foundLinkedHashedInos := len(linkedHashedInos) > 0
		if !foundLinkedHashedInos {
			// Get a list of previously seen inodes that may be linkable
			cachedSeq, useDigest := f.cachedInos(H, curPathStat)

			// Search the list of potential inode, looking for a match
			Stats.SearchedInoSeq()
			foundLinkable := false
			for _, cachedIno := range cachedSeq {
				Stats.IncInoSeqIterations()
				cachedPathStat := f.PathStatFromIno(cachedIno)
				if f.areFilesLinkable(cachedPathStat, curPathStat, useDigest) {
					f.addLinkableInos(cachedPathStat.Ino, curPathStat.Ino)
					foundLinkable = true
					break
				}
			}
			// Add hash to set if no match was found in current set
			if !foundLinkable {
				Stats.NoHashMatch()
				inoSet := f.InoHashes[H]
				inoSet.Add(statInfo.Ino)
				f.InoStatInfo[statInfo.Ino] = statInfo
			}
		}
	}
	// Remember Inode and filename/path information for each seen file
	f.InoStatInfo[statInfo.Ino] = statInfo
	f.InoAppendPathname(statInfo.Ino, curPath)
}

// possibleInos returns a slice of inos that can be searched for equal contents
func (f *FSDev) cachedInos(H Hash, ps PathStat) ([]Ino, bool) {
	var cachedSeq []Ino
	cachedSet := f.InoHashes[H]
	// If digests are enabled, and cached inode lists are
	// long enough, then switch on the use of digests.
	thresh := MyOptions.LinearSearchThresh
	useDigest := thresh >= 0 && len(cachedSet) > thresh
	if useDigest {
		digest, err := contentDigest(ps.Pathsplit.Join())
		if err == nil {
			// With digests, we take the (potentially long) set of cached inodes (ie.
			// those inodes that all have the same InoHash), and remove the inodes that
			// are definitely not a match because their digests do not match with the
			// current inode.  We also put the inodes with equal digests before those
			// that have no digest yet, in hopes of more quickly finding an identical file.
			f.addPathStatDigest(ps, digest)
			noDigests := cachedSet.Difference(f.InosWithDigest)
			sameDigests := cachedSet.Intersection(f.DigestIno[digest])
			differentDigests := cachedSet.Difference(sameDigests).Difference(noDigests)
			cachedSeq = append(sameDigests.AsSlice(), noDigests.AsSlice()...)

			PanicIf(noDigests.Has(ps.StatInfo.Ino), "New Ino found in noDigests\n")
			PanicIf(len(InoSetIntersection(sameDigests, differentDigests, noDigests)) > 0,
				"Overlapping digest sets\n")
		}
	} else {
		cachedSeq = cachedSet.AsSlice()
	}

	return cachedSeq, useDigest
}

func (f *FSDev) linkedInoSet(ino Ino) InoSet {
	if _, ok := f.LinkedInos[ino]; !ok {
		return NewInoSet(ino)
	}
	remainingInos := f.LinkedInosCopy()
	resultSet := NewInoSet()
	pending := append(make([]Ino, 0, 1), ino)
	for len(pending) > 0 {
		// Pop last item from pending as ino
		ino = pending[len(pending)-1]
		pending = pending[:len(pending)-1]

		// Add ino to results
		resultSet[ino] = exists
		// Add connected inos to pending
		if _, ok := remainingInos[ino]; ok {
			linkedInos := remainingInos[ino]
			delete(remainingInos, ino)
			linkedInoSlice := make([]Ino, len(linkedInos))
			i := 0
			for k := range linkedInos {
				linkedInoSlice[i] = k
				i++
			}
			pending = append(pending, linkedInoSlice...)
		}
	}
	return resultSet
}

func (f *FSDev) linkedInoSets() <-chan InoSet {
	out := make(chan InoSet)
	go func() {
		defer close(out)
		remainingInos := f.LinkedInosCopy()
		for startIno := range f.LinkedInos {
			if _, ok := remainingInos[startIno]; !ok {
				continue
			}
			resultSet := NewInoSet()
			pending := append(make([]Ino, 0, 1), startIno)
			for len(pending) > 0 {
				// Pop last item from pending as ino
				ino := pending[len(pending)-1]
				pending = pending[:len(pending)-1]

				// Add ino to results
				resultSet[ino] = exists
				// Add connected inos to pending
				if _, ok := remainingInos[ino]; ok {
					linkedInos := remainingInos[ino]
					delete(remainingInos, ino)
					linkedInoSlice := make([]Ino, len(linkedInos))
					i := 0
					for k := range linkedInos {
						linkedInoSlice[i] = k
						i++
					}
					pending = append(pending, linkedInoSlice...)
				}
			}
			out <- resultSet
		}
	}()
	return out
}

func (f *FSDev) ArbitraryPath(ino Ino) Pathsplit {
	// ino must exist in f.InoPaths.  If it does, there will be at least
	// one pathname to return
	filenamePaths := f.InoPaths[ino]
	return filenamePaths.any()
}

func (f *FSDev) ArbitraryFilenamePath(ino Ino, filename string) Pathsplit {
	filenamePaths := f.InoPaths[ino]
	return filenamePaths.anyWithFilename(filename)
}

func (f *FSDev) haveSeenPath(ino Ino, path Pathsplit) bool {
	fp := f.InoPaths[ino]
	return fp.hasPath(path)
}

func (f *FSDev) InoAppendPathname(ino Ino, path Pathsplit) {
	filenamePaths, ok := f.InoPaths[ino]
	if !ok {
		filenamePaths = newFilenamePaths()
		f.InoPaths[ino] = filenamePaths
	}
	filenamePaths.add(path)
}

func (f *FSDev) PathStatFromIno(ino Ino) PathStat {
	path := f.ArbitraryPath(ino)
	fi := f.InoStatInfo[ino]
	return PathStat{path, fi}
}

func (f *FSDev) allInoPaths(ino Ino) <-chan Pathsplit {
	// Deepcopy the FilenamePaths map so that we can update the original
	// while iterating over it's contents
	fpClone := f.InoPaths[ino].clone()

	// Iterate over the copy of the FilenamePaths, and return each pathname
	out := make(chan Pathsplit)
	go func() {
		defer close(out)
		for _, paths := range fpClone.pMap {
			for path := range paths {
				out <- path
			}
		}
	}()
	return out
}

func (f *FSDev) addLinkableInos(ino1, ino2 Ino) {
	// Add both src and destination inos to the linked InoSets
	inoSet1, ok := f.LinkedInos[ino1]
	if !ok {
		f.LinkedInos[ino1] = NewInoSet(ino2)
	} else {
		inoSet1.Add(ino2)
	}

	inoSet2, ok := f.LinkedInos[ino2]
	if !ok {
		f.LinkedInos[ino2] = NewInoSet(ino1)
	} else {
		inoSet2.Add(ino1)
	}
}

func (f *FSDev) areFilesLinkable(ps1 PathStat, ps2 PathStat, useDigest bool) bool {
	// Dev is equal for both PathStats
	if ps1.Ino == ps2.Ino {
		return false
	}
	if ps1.Size != ps2.Size {
		return false
	}
	if !MyOptions.IgnoreTime && !ps1.EqualTime(ps2) {
		return false
	}
	if !MyOptions.IgnorePerms && !ps1.EqualMode(ps2) {
		return false
	}
	if !MyOptions.IgnoreOwner && !ps1.EqualOwnership(ps2) {
		return false
	}
	if !MyOptions.IgnoreXattr {
		if eq, _ := equalXAttrs(ps1.Join(), ps2.Join()); !eq {
			return false
		}
	}

	// assert(st1.Dev == st2.Dev && st1.Ino != st2.Ino && st1.Size == st2.Size)
	if useDigest {
		f.newPathStatDigest(ps1)
		f.newPathStatDigest(ps2)
	}

	Stats.DidComparison()
	// error handling deferred
	eq, _ := areFileContentsEqual(ps1.Join(), ps2.Join())
	if eq {
		Stats.FoundEqualFiles()

		// Add some debugging statistics for files that are found to be
		// equal, but which have some mismatched inode parameters.
		addMismatchTotalBytes := false
		if !(ps1.Sec == ps2.Sec && ps1.Nsec == ps2.Nsec) {
			Stats.AddMismatchedMtimeBytes(ps1.Size)
			addMismatchTotalBytes = true
		}
		if ps1.Mode.Perm() != ps2.Mode.Perm() {
			Stats.AddMismatchedModeBytes(ps1.Size)
			addMismatchTotalBytes = true
		}
		if ps1.Uid != ps2.Uid {
			Stats.AddMismatchedUidBytes(ps1.Size)
			addMismatchTotalBytes = true
		}
		if ps1.Gid != ps2.Gid {
			Stats.AddMismatchedGidBytes(ps1.Size)
			addMismatchTotalBytes = true
		}
		eq, err := equalXAttrs(ps1.Join(), ps2.Join())
		if err == nil && !eq {
			Stats.AddMismatchedXattrBytes(ps1.Size)
			addMismatchTotalBytes = true
		}
		if addMismatchTotalBytes {
			Stats.AddMismatchedTotalBytes(ps1.Size)
		}
	}
	return eq
}

func (f *FSDev) moveLinkedPath(dstPath Pathsplit, srcIno Ino, dstIno Ino) {
	// Get pathnames slice matching Ino and filename
	fp := f.InoPaths[dstIno]
	fp.remove(dstPath)

	if fp.isEmpty() {
		delete(f.InoPaths, dstIno)
	}
	f.InoAppendPathname(srcIno, dstPath)
}

func (f *FSDev) addPathStatDigest(ps PathStat, digest Digest) {
	if !f.InosWithDigest.Has(ps.Ino) {
		f.helperPathStatDigest(ps, digest)
	}
}

func (f *FSDev) newPathStatDigest(ps PathStat) {
	if !f.InosWithDigest.Has(ps.Ino) {
		pathname := ps.Pathsplit.Join()
		digest, err := contentDigest(pathname)
		if err == nil {
			f.helperPathStatDigest(ps, digest)
		}
	}
}

func (f *FSDev) helperPathStatDigest(ps PathStat, digest Digest) {
	if _, ok := f.DigestIno[digest]; !ok {
		f.DigestIno[digest] = NewInoSet(ps.Ino)
	} else {
		set := f.DigestIno[digest]
		set.Add(ps.Ino)
	}
	f.InosWithDigest.Add(ps.Ino)
}
