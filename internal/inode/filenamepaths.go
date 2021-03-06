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

import P "github.com/chadnetzer/hardlinkable/internal/pathpool"

// Make a set for pathnames (instead of a slice)
type pathsplitSet map[P.Pathsplit]struct{}

func newPathsplitSet(vals ...P.Pathsplit) pathsplitSet {
	s := make(pathsplitSet)
	for _, v := range vals {
		s.add(v)
	}
	return s
}

func (p pathsplitSet) any() P.Pathsplit {
	for k := range p {
		return k
	}
	return P.Pathsplit{}
}

func (p pathsplitSet) add(ps P.Pathsplit) {
	p[ps] = struct{}{}
}

func (p pathsplitSet) remove(ps P.Pathsplit) {
	delete(p, ps)
}

func (p pathsplitSet) clone() pathsplitSet {
	c := make(pathsplitSet, len(p))
	for k := range p {
		c.add(k)
	}
	return c
}

// FilenamePaths holds a map of filenames to their full pathnames (ie. the
// different paths to an inode), and also holds an arbitrary pathname that can
// be used for consistency (rather than a fully random one from the map)
type FilenamePaths struct {
	FPMap   map[string]pathsplitSet // key = filename
	arbPath P.Pathsplit
}

func newFilenamePaths() *FilenamePaths {
	p := make(map[string]pathsplitSet)
	return &FilenamePaths{p, P.Pathsplit{}}
}

// When choosing an arbitrary pathname, remember what was chosen and return it
// consistently.  This prevents the source link paths from changing
// unnecessarily, and basically makes the output a bit more friendly.
func (f *FilenamePaths) Any() P.Pathsplit {
	if f.arbPath == (P.Pathsplit{}) {
		for _, pathnames := range f.FPMap {
			f.arbPath = pathnames.any()
			return f.arbPath
		}
	}
	return f.arbPath
}

// AnyWithFilename will return an arbitrary path with the given filename
func (f *FilenamePaths) AnyWithFilename(filename string) P.Pathsplit {
	if f.arbPath == (P.Pathsplit{}) || filename != f.arbPath.Filename {
		f.arbPath = f.FPMap[filename].any()
	}
	return f.arbPath
}

func (f *FilenamePaths) Add(ps P.Pathsplit) {
	p, ok := f.FPMap[ps.Filename]
	if !ok {
		p = newPathsplitSet()
	}
	p.add(ps)
	f.FPMap[ps.Filename] = p
}

func (f *FilenamePaths) Remove(ps P.Pathsplit) {
	// Find and remove given Pathsplit from FPMap
	f.FPMap[ps.Filename].remove(ps)
	if len(f.FPMap[ps.Filename]) == 0 {
		delete(f.FPMap, ps.Filename)
		f.arbPath = P.Pathsplit{}
	} else if ps == f.arbPath {
		f.arbPath = P.Pathsplit{}
	}
}

func (f *FilenamePaths) IsEmpty() bool {
	return len(f.FPMap) == 0
}

func (f *FilenamePaths) HasPath(ps P.Pathsplit) bool {
	paths, ok := f.FPMap[ps.Filename]
	if !ok {
		return false
	}
	if _, ok := paths[ps]; !ok {
		return false
	}
	return true
}

func (f *FilenamePaths) HasFilename(filename string) bool {
	_, ok := f.FPMap[filename]
	return ok
}

// CountPaths returns the number of stored paths
func (f *FilenamePaths) CountPaths() int {
	n := 0
	for _, paths := range f.FPMap {
		n += len(paths)
	}
	return n
}

// PathsAsSlice returns a slice of all the stored paths
func (f *FilenamePaths) PathsAsSlice() []P.Pathsplit {
	// Makes two passes over the FilenamePaths maps in order to preallocate
	// and fill the slice.  Not clear if its an actual time saver over
	// appends...
	s := make([]P.Pathsplit, f.CountPaths())
	i := 0
	for _, paths := range f.FPMap {
		for path := range paths {
			s[i] = path
			i++
		}
	}
	return s
}
