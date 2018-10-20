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
	P "github.com/chadnetzer/hardlinkable/internal/pathpool"
)

type PathInfo struct {
	P.Pathsplit
	StatInfo
}

func (p1 PathInfo) EqualTime(p2 PathInfo) bool {
	return p1.Mtim.Equal(p2.Mtim)
}

func (p1 PathInfo) EqualMode(p2 PathInfo) bool {
	return p1.Mode == p2.Mode
}

func (p1 PathInfo) EqualOwnership(p2 PathInfo) bool {
	return p1.Uid == p2.Uid && p1.Gid == p2.Gid
}

type PathsMap map[Ino]*FilenamePaths

func (pm PathsMap) ArbitraryPath(ino Ino) P.Pathsplit {
	// ino must exist in f.InoPaths.  If it does, there will be at least
	// one pathname to return
	return pm[ino].Any()
}

func (pm PathsMap) ArbitraryFilenamePath(ino Ino, filename string) P.Pathsplit {
	return pm[ino].AnyWithFilename(filename)
}

func (pm PathsMap) HasPath(ino Ino, path P.Pathsplit) bool {
	return pm[ino].HasPath(path)
}

func (pm PathsMap) AppendPath(ino Ino, path P.Pathsplit) {
	fp, ok := pm[ino]
	if !ok {
		fp = newFilenamePaths()
		pm[ino] = fp
	}
	fp.Add(path)
}

// AllPaths returns a channel that can be iterated over to sequentially access
// all the paths for a given inode.
func (pm PathsMap) AllPaths(ino Ino) <-chan P.Pathsplit {
	// To avoid concurrent modification to the PathsMap maps while
	// iterating from another goroutine, first place all the pathnames into
	// a slice, in order to send them over the channel.
	paths := pm[ino].PathsAsSlice()

	out := make(chan P.Pathsplit)
	go func() {
		defer close(out)
		for _, path := range paths {
			out <- path
		}
	}()
	return out
}

// MovePath moves the given destination path, from the given destination inode,
// to the source inode.
func (pm PathsMap) MovePath(dstPath P.Pathsplit, srcIno Ino, dstIno Ino) {
	// Get pathnames slice matching Ino and filename
	fp := pm[dstIno]
	fp.Remove(dstPath)

	if fp.IsEmpty() {
		delete(pm, dstIno)
	}
	pm.AppendPath(srcIno, dstPath)
}

// PathCount returns the number of unique paths and dirs encountered after the
// initial walk is completed.  This can give us an accurate count of the number
// of inode nlinks we should encounter if all linked paths are included in the
// walk.  Conversely, if we count the nlinks from all the encountered inodes,
// and compare to the number of paths this function returns, we should have a
// count of how many inode paths were not seen by the walk.
func (pm PathsMap) PathCount() (paths int64, dirs int64) {
	var numPaths, numDirs int64

	// Make a set for storing unique dirs (key = dirname)
	dirMap := make(map[string]struct{})

	// loop over all inos, getting FilenamePaths
	for _, fp := range pm {
		// loop over all filenames, getting paths
		for _, paths := range fp.FPMap {
			// Loop over all paths
			for p := range paths {
				numPaths++
				dirMap[p.Dirname] = struct{}{}
			}
		}
		// Count the number of unique dirs and increment
	}
	numDirs = int64(len(dirMap))

	return numPaths, numDirs
}
