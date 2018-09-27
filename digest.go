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
	"hash/fnv"
	"os"
)

type digestVal uint32

type inoDigests struct {
	InoSets        map[digestVal]I.Set
	InosWithDigest I.Set
}

func newInoDigests() inoDigests {
	return inoDigests{
		InoSets:        make(map[digestVal]I.Set),
		InosWithDigest: I.NewSet(),
	}
}

func (id *inoDigests) getInos(d digestVal) I.Set {
	return id.InoSets[d]
}

func (id *inoDigests) add(pi I.PathInfo, digest digestVal) {
	if !id.InosWithDigest.Has(pi.Ino) {
		digestHelper(id, pi, digest)
	}
}

func (id *inoDigests) newDigest(pi I.PathInfo) error {
	var err error
	if !id.InosWithDigest.Has(pi.Ino) {
		pathname := pi.Pathsplit.Join()
		digest, err := contentDigest(pathname)
		if err == nil {
			digestHelper(id, pi, digest)
		}
	}
	return err
}

func digestHelper(id *inoDigests, pi I.PathInfo, digest digestVal) {
	if _, ok := id.InoSets[digest]; !ok {
		id.InoSets[digest] = I.NewSet(pi.Ino)
	} else {
		set := id.InoSets[digest]
		set.Add(pi.Ino)
	}
	id.InosWithDigest.Add(pi.Ino)
}

// Return a short digest of the first part of the given pathname, to help
// determine if two files are definitely not equivalent, without doing a full
// comparison.  Typically this will be used when a full file comparison will be
// performed anyway (incurring the IO overhead), and saving the digest to help
// quickly reduce the set of possibly equal inodes later (ie. reducing the
// length of the repeated linear searches).
func contentDigest(r *Results, pathname string) (digestVal, error) {
	const bufSize = 8192

	f, err := os.Open(pathname)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, bufSize)
	_, err = f.Read(buf)
	if err != nil {
		return 0, err
	}

	r.computedDigest()

	hash := fnv.New32a()
	hash.Write(buf)
	return digestVal(hash.Sum32()), nil
}
