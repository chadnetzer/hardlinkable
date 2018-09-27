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
	P "hardlinkable/internal/pathpool"
)

type PathInfo struct {
	P.Pathsplit
	StatInfo
}

func (p1 PathInfo) EqualTime(p2 PathInfo) bool {
	return p1.Sec == p2.Sec && p1.Nsec == p2.Nsec
}

func (p1 PathInfo) EqualMode(p2 PathInfo) bool {
	return p1.Mode.Perm() == p2.Mode.Perm()
}

func (p1 PathInfo) EqualOwnership(p2 PathInfo) bool {
	return p1.Uid == p2.Uid && p1.Gid == p2.Gid
}

type PathsMap map[Ino]*filenamePaths

func (ip PathsMap) ArbitraryPath(ino Ino) P.Pathsplit {
	// ino must exist in f.InoPaths.  If it does, there will be at least
	// one pathname to return
	return ip[ino].Any()
}

func (ip PathsMap) ArbitraryFilenamePath(ino Ino, filename string) P.Pathsplit {
	return ip[ino].AnyWithFilename(filename)
}

func (ip PathsMap) HasPath(ino Ino, path P.Pathsplit) bool {
	return ip[ino].HasPath(path)
}

func (ip PathsMap) AppendPath(ino Ino, path P.Pathsplit) {
	fp, ok := ip[ino]
	if !ok {
		fp = newFilenamePaths()
		ip[ino] = fp
	}
	fp.Add(path)
}

// AllPaths returns a channel that can be iterated over to sequentially access
// all the paths for a given inode.
//
// Note - it does *not* make a copy of the maps being iterated over, so if a
// not yet visited path is removed from the map it won't be sent over the
// channel, and new paths added to the map may or may not be sent over the
// channel. (ie. standard Go map semantics)
//
// Note that this method is used with MovePath, which does remove and add paths
// to the same inner map.  But, this is okay since we are iterating over
// destination paths, and it'll remove an already seen path from the
// destination inode inner map (which won't affect the remaining destination
// paths being sent over the channel), and add it to the src inode (which is a
// different inode map entry).  So no clone of the filenamepaths maps are
// needed.
func (ip PathsMap) AllPaths(ino Ino) <-chan P.Pathsplit {
	// Iterate over the copy of the FilenamePaths, and return each pathname
	out := make(chan P.Pathsplit)
	go func() {
		defer close(out)
		for _, paths := range ip[ino].PMap {
			for path := range paths {
				out <- path
			}
		}
	}()
	return out
}

func (ip PathsMap) MovePath(dstPath P.Pathsplit, srcIno Ino, dstIno Ino) {
	// Get pathnames slice matching Ino and filename
	fp := ip[dstIno]
	fp.Remove(dstPath)

	if fp.IsEmpty() {
		delete(ip, dstIno)
	}
	ip.AppendPath(srcIno, dstPath)
}

// PathCount returns the number of unique paths and dirs encountered after the
// initial walk is completed.  This can give us an accurate count of the number
// of inode nlinks we should encounter if all linked paths are included in the
// walk.  Conversely, if we count the nlinks from all the encountered inodes,
// and compare to the number of paths this function returns, we should have a
// count of how many inode paths were not seen by the walk.
func (ip PathsMap) PathCount() (paths int64, dirs int64) {
	var numPaths, numDirs int64

	// Make a set for storing unique dirs
	dirMap := make(map[string]struct{})

	// loop over all inos, getting FilenamePaths
	for _, fp := range ip {
		// loop over all filenames, getting paths
		for _, paths := range fp.PMap {
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
