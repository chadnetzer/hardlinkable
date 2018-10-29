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

package inode

import (
	"hash/fnv"
	"io"
	"os"
)

type Digest uint32

type InoDigests struct {
	InoSets        map[Digest]Set
	InosWithDigest Set
}

func NewInoDigests() InoDigests {
	return InoDigests{
		InoSets:        make(map[Digest]Set),
		InosWithDigest: NewSet(),
	}
}

func (id *InoDigests) GetInos(d Digest) Set {
	return id.InoSets[d]
}

func (id *InoDigests) Add(pi PathInfo, digest Digest) {
	if !id.InosWithDigest.Has(pi.Ino) {
		digestHelper(id, pi, digest)
	}
}

func (id *InoDigests) NewDigest(pi PathInfo, buf []byte) bool {
	var computed bool
	if !id.InosWithDigest.Has(pi.Ino) {
		pathname := pi.Pathsplit.Join()
		digest, err := ContentDigest(pathname, buf)
		if err == nil {
			digestHelper(id, pi, digest)
			computed = true
		}
	}
	return computed
}

func digestHelper(id *InoDigests, pi PathInfo, digest Digest) {
	if _, ok := id.InoSets[digest]; !ok {
		id.InoSets[digest] = NewSet(pi.Ino)
	} else {
		set := id.InoSets[digest]
		set.Add(pi.Ino)
	}
	id.InosWithDigest.Add(pi.Ino)
}

// ContentDigest returns a short digest of the first part of the given
// pathname, to help determine if two files are definitely not equivalent,
// without doing a full comparison.  Typically this will be used when a full
// file comparison will be performed anyway (incurring the IO overhead), and
// saving the digest to help quickly reduce the set of possibly equal inodes
// later (ie. reducing the length of the repeated linear searches).
func ContentDigest(pathname string, buf []byte) (Digest, error) {
	f, err := os.Open(pathname)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := ReadChunk(f, buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n < len(buf) {
		buf = buf[:n]
	}

	hash := fnv.New32a()
	_, err = hash.Write(buf)
	if err != nil {
		return 0, err
	}
	return Digest(hash.Sum32()), nil
}
