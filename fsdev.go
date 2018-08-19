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
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
)

type Ino = uint64
type Hash uint64
type Digest uint32

type InoSet map[Ino]struct{}

type FileInfos map[string]os.FileInfo

type FileNamePairs map[string][]Namepair

type PathStat struct {
	Namepair Namepair
	FileInfo os.FileInfo
}

type FSDev struct {
	Dev            int64
	MaxNLinks      uint64
	InoHashes      map[Hash]InoSet
	InoFileInfo    map[Ino]os.FileInfo
	InoNamepairs   map[Ino]FileNamePairs
	LinkedInos     map[Ino]InoSet
	DigestIno      map[Digest]InoSet
	InosWithDigest InoSet

	// For each directory name, keep track of all the FileInfo structures
	DirnameFileInfos map[string]FileInfos
}

var exists = struct{}{}

func NewInoSet(inos ...Ino) InoSet {
	set := make(map[Ino]struct{})
	for _, ino := range inos {
		set[ino] = exists
	}
	return set
}

func (i *InoSet) Add(ino Ino) {
	(*i)[ino] = exists
}

func (i *InoSet) Copy() InoSet {
	newSet := NewInoSet()
	for k,_ := range *i {
		newSet[k] = exists
	}
	return newSet
}

func (i *InoSet) Intersection(set2 InoSet) InoSet {
	resultSet := NewInoSet()
	var little, big *InoSet
	if len(*i) <= len(set2) {
		little, big = i, &set2
	} else {
		little, big = &set2, i
	}
	for k, _ := range *little {
		if _, ok := (*big)[k]; ok {
			resultSet[k] = exists
		}
	}
	return resultSet
}

func (f *FSDev) LinkedInosCopy() map[Ino]InoSet {
	newLinkedInos := make(map[Ino]InoSet)
	for k,v := range f.LinkedInos {
		newLinkedInos[k] = v.Copy()
	}
	return newLinkedInos
}

func NewFSDev(dev int64) FSDev {
	var w FSDev
	w.Dev = dev
	w.InoHashes = make(map[Hash]InoSet)
	w.InoFileInfo = make(map[Ino]os.FileInfo)
	w.InoNamepairs = make(map[Ino]FileNamePairs)
	w.LinkedInos = make(map[Ino]InoSet)
	w.DigestIno = make(map[Digest]InoSet)
	w.InosWithDigest = NewInoSet()

	return w
}

// Produce an equal hash for potentially equal files, based only on Inode
// metadata (size, time, etc.)
func InoHash(stat syscall.Stat_t, opt Options) Hash {
	var value Hash
	size := Hash(stat.Size)
	mtim := stat.Mtimespec
	// The main requirement is that files that could be equal have equal
	// hashes.  It's less important if unequal files also have the same
	// hash value, since we will still compare the actual file content
	// later.
	if opt.IgnoreTime || opt.ContentOnly {
		value = size
	} else {
		value = size ^ Hash(mtim.Sec) ^ Hash(mtim.Nsec)
	}
	return value
}

func (f *FSDev) findIdenticalFiles(pathname string, fileInfo os.FileInfo) {
	sysStat := *fileInfo.Sys().(*syscall.Stat_t)
	//fmt.Println("pathname: ", pathname)
	dirname, filename := path.Split(pathname)
	namepair := Namepair{ dirname, filename }
	pathStat := PathStat { namepair, fileInfo }

	if _, ok := f.InoFileInfo[sysStat.Ino]; !ok {
		//fmt.Println("find inode: ", pathname, sysStat.Ino)
		Stats.FoundInode()
	}

	inoHash := InoHash(sysStat, MyOptions)
	//fmt.Println("hash and inode: ", inoHash, sysStat.Ino)
	if _, ok := f.InoHashes[inoHash]; !ok {
		Stats.MissedHash()
		f.InoHashes[inoHash] = NewInoSet(sysStat.Ino)
		//fmt.Println("new inode set: ", inoHash, sysStat.Ino, f.InoHashes[inoHash])
	} else {
		Stats.FoundHash()
		if _, ok := f.InoFileInfo[sysStat.Ino]; ok {
			prevNamepair := f.ArbitraryNamepair(sysStat.Ino)
			prevFileinfo := f.InoFileInfo[sysStat.Ino]
			existingLinkInfo := ExistingLink{ prevNamepair, namepair, prevFileinfo }
			Stats.FoundExistingHardlink(existingLinkInfo)
			//fmt.Println("prevNamepair: ", prevNamepair, prevFileinfo)
		}
		linkedInos := f.linkedInoSet(sysStat.Ino)
		//fmt.Printf("linkedInos %+v\n", linkedInos)
		hashedInos := f.InoHashes[inoHash]
		//fmt.Printf("hashedInos %+v\n", hashedInos)
		linkedHashedInos := linkedInos.Intersection(hashedInos)
		//fmt.Printf("linkedHashedInos %+v\n", linkedHashedInos)
		foundLinkedHashedInos := len(linkedHashedInos) > 0
		if !foundLinkedHashedInos {
			Stats.SearchedInoSeq()
			cachedInoSet := f.InoHashes[inoHash]
			loopEndedEarly := false
			for cachedIno := range cachedInoSet {
				Stats.IncInoSeqIterations()
				cachedPathStat := f.PathStatFromIno(cachedIno)
				if areFilesHardlinkable(cachedPathStat, pathStat) {
					//fmt.Println("Matching files: ", pathStat, cachedPathStat)
					loopEndedEarly = true
					f.foundHardlinkableFiles(cachedPathStat, pathStat)
					break
				}
			}
			if !loopEndedEarly {
				Stats.NoHashMatch()
				inoSet := f.InoHashes[inoHash]
				inoSet.Add(sysStat.Ino)
				f.InoFileInfo[sysStat.Ino] = fileInfo
			}
		}
	}
	f.InoFileInfo[sysStat.Ino] = fileInfo
	f.InoAppendPathname(sysStat.Ino, pathname)
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
		ino = pending[len(pending) - 1]
		pending = pending[:len(pending) - 1]

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

func (f *FSDev) ArbitraryNamepair(ino Ino) Namepair {
	m := f.InoNamepairs[ino]
	var v []Namepair
	for _, v = range m {
		return v[0]
	}
	return Namepair{ "foo", "bar" }
}

func (f *FSDev) InoAppendPathname(ino Ino, pathname string) {
	dirname, filename := path.Split(pathname)
	namepair := Namepair{ dirname, filename }
	fileNamePairs, ok := f.InoNamepairs[ino]
	if !ok {
		fileNamePairs = make(FileNamePairs)
	}
	var namepairs []Namepair
	namepairs, ok = fileNamePairs[filename]
	if !ok  {
		namepairs = make([]Namepair, 0)
	}
	namepairs = append(namepairs, namepair)
	fileNamePairs[filename] = namepairs
	f.InoNamepairs[ino] = fileNamePairs

	//fmt.Println("filenamepairs ", fileNamePairs)
}

func (f *FSDev) PathStatFromIno(ino Ino) PathStat {
	namepair := f.ArbitraryNamepair(ino)
	fi := f.InoFileInfo[ino]
	return PathStat { namepair, fi }
}

func (f *FSDev) foundHardlinkableFiles(fs1, fs2 PathStat) {
	ino1 := fs1.FileInfo.Sys().(*syscall.Stat_t).Ino
	ino2 := fs2.FileInfo.Sys().(*syscall.Stat_t).Ino

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
	Stats.FoundHardlinkableFiles(fs1.Namepair, fs2.Namepair)
}

func areFilesHardlinkable(ps1 PathStat, ps2 PathStat) bool {
	st1 := ps1.FileInfo.Sys().(*syscall.Stat_t)
	st2 := ps2.FileInfo.Sys().(*syscall.Stat_t)
	if st1.Dev != st2.Dev {
		return false
	}
	if st1.Ino == st2.Ino {
		return false
	}
	if st1.Size != st2.Size {
		return false
	}
	// Add options checking later (time/perms/ownership/etc)

	// assert(st1.Dev == st2.Dev && st1.Ino != st2.Ino && st1.Size == st2.Size)
	pathname1 := path.Join(ps1.Namepair.Dirname, ps1.Namepair.Filename)
	pathname2 := path.Join(ps2.Namepair.Dirname, ps2.Namepair.Filename)

	Stats.DidComparison()
	// error handling deferred
	eq, _ := areFileContentsEqual(pathname1, pathname2)
	if eq {
		Stats.FoundEqualFiles()
	}
	return eq
}

func areFileContentsEqual(pathname1, pathname2 string) (bool, error){
	var openErr error
	f1, openErr := os.Open(pathname1)
	if openErr != nil {
		return false, openErr
	}
	defer f1.Close()

	f2, openErr := os.Open(pathname2)
	if openErr != nil {
		return false, openErr
	}
	defer f2.Close()

	eq, err := cmpReaderContents(f1, f2)
	if eq {
		fmt.Println("Matched: ", pathname1, pathname2)
	}
	return eq, err
}

// Return true if r1 and r2 have identical contents. Otherwise return false.
func cmpReaderContents(r1, r2 io.Reader) (bool, error) {
	const bufSize = 8192
	buf1 := make([]byte, bufSize)
	buf2 := make([]byte, bufSize)

	for {
		n1, err1 := r1.Read(buf1)
		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		n2, err2 := r2.Read(buf2)
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}

		if n1 != n2 {
			return false, nil
		}

		if n1 == 0 {
			return true, nil
		}

		if bytes.Compare(buf1, buf2) != 0 {
			return false, nil
		}
	}
	return false, nil
}
