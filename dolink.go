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
	"os"
)

func (fs *fsDev) hardlinkFiles(src, dst I.PathInfo) error {
	// Quit early if the src or dst files have changed since we first
	// stat()-ed them.
	if hasBeenModified(src, fs.Dev) {
		return fmt.Errorf("Detected modified file before linking: %v", src.Pathsplit.Join())
	}
	if hasBeenModified(dst, fs.Dev) {
		return fmt.Errorf("Detected modified file before linking: %v", dst.Pathsplit.Join())
	}

	tmpName := dst.Pathsplit.Join() + ".tmp_while_linking"
	if err := os.Link(src.Pathsplit.Join(), tmpName); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst.Pathsplit.Join()); err != nil {
		os.Remove(tmpName)
		return err
	}

	// Use destination file times if it's most recently modified
	dstTime := dst.MTime()
	if dstTime.After(src.MTime()) {
		err := os.Chtimes(src.Pathsplit.Join(), dstTime, dstTime)
		if err != nil {
			// Ignore this error, and just return early, as we
			// don't want to abort the Run().  Any error returned
			// from this function is considered fatal.
			return nil
		}

		// Ignore failure if we can't chown the inode
		os.Chown(src.Pathsplit.Join(), int(src.Uid), int(src.Gid))

		// Keep our cached inode.Info time updated
		si := fs.InoStatInfo[src.Ino]
		si.Sec = dst.Sec
		si.Nsec = dst.Nsec
		fs.InoStatInfo[src.Ino] = si
	}
	return nil
}

func hasBeenModified(pi I.PathInfo, dev uint64) bool {
	newDSI, err := I.LInfo(pi.Pathsplit.Join())
	if err != nil {
		return true
	}

	if newDSI.Dev != dev ||
		newDSI.Ino != pi.Ino ||
		newDSI.Size != pi.Size ||
		newDSI.Sec != pi.Sec ||
		newDSI.Nsec != pi.Nsec ||
		newDSI.Mode != pi.Mode ||
		newDSI.Uid != pi.Uid ||
		newDSI.Gid != pi.Gid {
		return true
	}
	return false
}
