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

package linkable

import (
	"hardlinkable/options"
	"os"
	"syscall"
)

type InoSet map[int64]struct{}

type FileInfos map[string]os.FileInfo

type NamePair struct {
	Dirname  string
	Filename string
}

type FileNamePaths map[string][]NamePair

type FSDev struct {
	Dev            int64
	MaxNLinks      uint64
	InoHashes      map[uint64]InoSet
	InoFileInfo    map[uint64]os.FileInfo
	InoPathnames   map[uint64][]string
	LinkedInos     map[uint64]InoSet
	DigestIno      map[uint64]InoSet
	InosWithDigest InoSet

	// For each directory name, keep track of all the FileInfo structures
	DirnameFileInfos map[string]FileInfos
}

func NewInoSet() InoSet {
	return make(map[int64]struct{})
}

func NewFSDev(dev int64) FSDev {
	var w FSDev
	w.Dev = dev
	w.InoHashes = make(map[uint64]InoSet)
	w.InoFileInfo = make(map[uint64]os.FileInfo)
	w.InoPathnames = make(map[uint64][]string)
	w.LinkedInos = make(map[uint64]InoSet)
	w.DigestIno = make(map[uint64]InoSet)
	w.InosWithDigest = NewInoSet()

	return w
}

// Produce an equal hash for potentially equal files, based only on Inode
// metadata (size, time, etc.)
func InoHash(stat syscall.Stat_t, opt options.Options) uint64 {
	var value uint64
	size := uint64(stat.Size)
	mtim := stat.Mtimespec
	// The main requirement is that files that could be equal have equal
	// hashes.  It's less important if unequal files also have the same
	// hash value, since we will still compare the actual file content
	// later.
	if opt.IgnoreTime || opt.ContentOnly {
		value = size
	} else {
		value = size ^ uint64(mtim.Sec) ^ uint64(mtim.Nsec)
	}
	return value
}
