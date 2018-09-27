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

import P "hardlinkable/internal/pathpool"

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
	c := newPathsplitSet()
	for k, _ := range p {
		c.add(k)
	}
	return c
}

// filenamePaths holds a map of filenames to their full pathnames (ie. the
// different paths to an inode), and also holds an arbitrary pathname that can
// be used for consistency (rather than a fully random one from the map)
type filenamePaths struct {
	PMap    map[string]pathsplitSet
	arbPath P.Pathsplit
}

func newFilenamePaths() *filenamePaths {
	p := make(map[string]pathsplitSet)
	return &filenamePaths{p, P.Pathsplit{}}
}

// When choosing an arbitrary pathname, remember what was chosen and return it
// consistently.  This prevents the source link paths from changing
// unnecessarily, and basically makes the output a bit more friendly.
func (f *filenamePaths) Any() P.Pathsplit {
	if f.arbPath == (P.Pathsplit{}) {
		for _, pathnames := range f.PMap {
			f.arbPath = pathnames.any()
			return f.arbPath
		}
	}
	return f.arbPath
}

// AnyWithFilename will return an arbitrary path with the given filename
func (f *filenamePaths) AnyWithFilename(filename string) P.Pathsplit {
	f.arbPath = f.PMap[filename].any()
	return f.arbPath
}

func (f *filenamePaths) Add(ps P.Pathsplit) {
	p, ok := f.PMap[ps.Filename]
	if !ok {
		p = newPathsplitSet()
	}
	p.add(ps)
	f.PMap[ps.Filename] = p
}

func (f *filenamePaths) Remove(ps P.Pathsplit) {
	// Find and remove given Pathsplit from PMap
	f.PMap[ps.Filename].remove(ps)
	if len(f.PMap[ps.Filename]) == 0 {
		delete(f.PMap, ps.Filename)
		f.arbPath = P.Pathsplit{}
	} else if ps == f.arbPath {
		f.arbPath = P.Pathsplit{}
	}
}

func (f *filenamePaths) IsEmpty() bool {
	return len(f.PMap) == 0
}

func (f *filenamePaths) HasPath(ps P.Pathsplit) bool {
	paths, ok := f.PMap[ps.Filename]
	if !ok {
		return false
	}
	if _, ok := paths[ps]; !ok {
		return false
	}
	return true
}

// Return a copy of the given filenamePaths
func (f *filenamePaths) Copy() *filenamePaths {
	c := make(map[string]pathsplitSet, len(f.PMap))
	for k, v := range f.PMap {
		c[k] = v.clone()
	}
	return &filenamePaths{c, f.arbPath}
}
