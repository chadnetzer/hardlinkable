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

// Make a set for pathnames (instead of a slice)
type pathsplitSet map[Pathsplit]struct{}

func newPathsplitSet(vals ...Pathsplit) pathsplitSet {
	s := make(pathsplitSet)
	for _, v := range vals {
		s.add(v)
	}
	return s
}

func (p pathsplitSet) any() Pathsplit {
	for k := range p {
		return k
	}
	return Pathsplit{}
}

func (p pathsplitSet) add(ps Pathsplit) {
	p[ps] = struct{}{}
}

func (p pathsplitSet) remove(ps Pathsplit) {
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
	pMap    map[string]pathsplitSet
	arbPath Pathsplit
}

func newFilenamePaths() filenamePaths {
	p := make(map[string]pathsplitSet)
	return filenamePaths{p, Pathsplit{}}
}

// When choosing an arbitrary pathname, remember what was chosen and return it
// consistently.  This prevents the source link paths from changing
// unnecessarily, and basically makes the output a bit more friendly.
func (f filenamePaths) any() Pathsplit {
	if f.arbPath == (Pathsplit{}) {
		for _, pathnames := range f.pMap {
			f.arbPath = pathnames.any()
			return f.arbPath
		}
	}
	return f.arbPath
}

// anyWithFilename will return an arbitrary path with the given filename
func (f filenamePaths) anyWithFilename(filename string) Pathsplit {
	f.arbPath = f.pMap[filename].any()
	return f.arbPath
}

func (f *filenamePaths) add(ps Pathsplit) {
	p, ok := f.pMap[ps.Filename]
	if !ok {
		p = newPathsplitSet()
	}
	p.add(ps)
	f.pMap[ps.Filename] = p
}

func (f *filenamePaths) remove(ps Pathsplit) {
	// Find and remove given Pathsplit from pMap
	f.pMap[ps.Filename].remove(ps)
	if len(f.pMap) == 0 {
		delete(f.pMap, ps.Filename)
		f.arbPath = Pathsplit{}
	}
}

func (f filenamePaths) isEmpty() bool {
	return len(f.pMap) == 0
}

// Return a copy of the given filenamePaths
func (f filenamePaths) clone() filenamePaths {
	c := make(map[string]pathsplitSet, len(f.pMap))
	for k, v := range f.pMap {
		c[k] = v.clone()
	}
	return filenamePaths{c, f.arbPath}
}
