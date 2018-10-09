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
	"fmt"
	I "hardlinkable/internal/inode"
	"math/rand"
	"os"
	"strconv"
)

// haveNotBeenModified returns an error if a given PathInfo has changed on disk
func (fs *fsDev) haveNotBeenModified(paths ...I.PathInfo) error {
	for _, p := range paths {
		if hasBeenModified(p, fs.Dev) {
			return fmt.Errorf("Detected modified file before linking: %v", p.Pathsplit.Join())
		}
	}
	return nil
}

// hardlinkFiles() will unconditionally attempt link dst (ie. target) to src
func (fs *fsDev) hardlinkFiles(src, dst I.PathInfo) error {
	// Add some randomness to the tmpName to minimize chances of collisions
	// with deliberately targeted matching names
	tmpName := dst.Pathsplit.Join() + ".tmp" + strconv.FormatUint(rand.Uint64(), 36)
	if err := os.Link(src.Pathsplit.Join(), tmpName); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst.Pathsplit.Join()); err != nil {
		os.Remove(tmpName)
		return err
	}

	if fs.Options.UseNewestLink {
		// Use destination file times if it's most recently modified
		dstTime := dst.Mtim
		if dstTime.After(src.Mtim) {
			err := os.Chtimes(src.Pathsplit.Join(), dstTime, dstTime)
			if err != nil {
				fs.Results.FailedLinkChtimesCount++
				// Ignore this error, and just return early, as we
				// don't want to abort the Run().
				return nil
			}

			// Keep cached inode.StatInfo time updated
			si := fs.inoStatInfo[src.Ino]
			si.Mtim = dst.Mtim

			// Change uid/gid if possible
			err = os.Lchown(src.Pathsplit.Join(), int(src.Uid), int(src.Gid))
			if err != nil {
				fs.Results.FailedLinkChownCount++
				return nil
			}
			// Chown succeeded, so update the cached stat structures
			si.Uid = dst.Uid
			si.Gid = dst.Gid
		}
	}
	return nil
}

func hasBeenModified(pi I.PathInfo, dev uint64) bool {
	newDSI, err := I.LStatInfo(pi.Pathsplit.Join())
	if err != nil {
		return true
	}

	if newDSI.Dev != dev ||
		newDSI.Ino != pi.Ino ||
		newDSI.Nlink != pi.Nlink ||
		newDSI.Size != pi.Size ||
		!newDSI.Mtim.Equal(pi.Mtim) ||
		newDSI.Mode != pi.Mode ||
		newDSI.Uid != pi.Uid ||
		newDSI.Gid != pi.Gid {
		return true
	}
	return false
}
