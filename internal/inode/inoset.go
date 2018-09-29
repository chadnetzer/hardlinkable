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

import "sort"

type Ino = uint64

type Set map[Ino]struct{}

var exists = struct{}{}

// Return a non-nil Set with the optional inos in it
func NewSet(inos ...Ino) Set {
	set := make(map[Ino]struct{}, len(inos))
	for _, ino := range inos {
		set[ino] = exists
	}
	return set
}

// Add an Ino to the Set
func (s Set) Add(ino Ino) {
	s[ino] = exists
}

// Remove an Ino to the Set
func (s Set) Remove(ino Ino) {
	delete(s, ino)
}

// Return true if given Ino is in the Set
func (s Set) Has(ino Ino) bool {
	_, ok := s[ino]
	return ok
}

// Return true if all given Inos are in the Set (false if empty)
func (s Set) HasAll(inos ...Ino) bool {
	if len(inos) == 0 || len(s) == 0 {
		return false
	}
	for _, ino := range inos {
		if _, ok := s[ino]; !ok {
			return false
		}
	}
	return true
}

// Return a duplicate of the Set
func (s Set) Copy() Set {
	newSet := make(map[Ino]struct{}, len(s))
	for k := range s {
		newSet[k] = exists
	}
	return newSet
}

// Return an intersection of the receiver with a Set
func (s Set) Intersection(set2 Set) Set {
	resultSet := NewSet()
	var little, big *Set
	// Iterate over smaller set
	if len(s) <= len(set2) {
		little, big = &s, &set2
	} else {
		little, big = &set2, &s
	}
	for k := range *little {
		if _, ok := (*big)[k]; ok {
			resultSet[k] = exists
		}
	}
	return resultSet
}

// Return an intersection of multiple Sets
func SetIntersections(sets ...Set) Set {
	if len(sets) == 0 {
		return NewSet()
	}

	resultSet := sets[0].Copy()
	set := sets[0]
	for _, other := range sets {
		resultSet = set.Intersection(other)
		set = resultSet
	}
	return resultSet
}

// Return a difference of the other Set from the receiver
func (s Set) Difference(other Set) Set {
	// Iterate over smaller set
	resultSet := NewSet()
	for k := range s {
		if _, ok := other[k]; !ok {
			resultSet.Add(k)
		}
	}
	return resultSet
}

// Return the content of Set as a slice
func (s Set) AsSlice() []Ino {
	r := make([]Ino, len(s))
	i := 0
	for k := range s {
		r[i] = k
		i++
	}
	return r
}

type LinkableInoSets map[Ino]Set

// Add places both ino1 and ino2 into the LinkableInoSets map.
//
// Potentially races with All(), but typically all the data is collected and
// added with AddLinkableInos() before calling All() (so we don't bother with
// locking).
func (l LinkableInoSets) Add(ino1, ino2 Ino) {
	// Add both src and destination inos to the linkable InoSets
	inoSet1, ok := l[ino1]
	if !ok {
		l[ino1] = NewSet(ino2)
	} else {
		inoSet1.Add(ino2)
	}

	inoSet2, ok := l[ino2]
	if !ok {
		l[ino2] = NewSet(ino1)
	} else {
		inoSet2.Add(ino1)
	}
}

// linkableInoSetHelper is used by Containing and All to iterate over the
// LinkableInos map to return a connected set of inodes (ie.  inodes that the
// hardlinkable algorithm has determined are allowed to be linked together.)
func linkableInoSetHelper(l LinkableInoSets, ino Ino, seen Set) Set {
	results := NewSet(ino)
	pending := NewSet(ino)
	for len(pending) > 0 {
		// Pop item from pending set
		for ino = range pending {
			break
		}
		pending.Remove(ino)
		results.Add(ino)

		// Don't check for linkable inos that we've seen already
		if seen.Has(ino) {
			continue
		}
		seen.Add(ino)

		// Add connected inos to pending
		if linkable, ok := l[ino]; ok {
			for k := range linkable {
				pending.Add(k)
			}
		}
	}
	return results
}

// Containing calls linkableInoSetHelper to return a single set of linkable
// inodes containing the given 'ino'.  Linkable inodes are those determined by
// the algorithm to have been able to be hardlinked together (ie. have
// identical contents, and compatible inode parameters)
func (l LinkableInoSets) Containing(ino Ino) Set {
	if _, ok := l[ino]; !ok {
		return NewSet(ino)
	}
	seen := NewSet()
	return linkableInoSetHelper(l, ino, seen)
}

// All sends all the linkable InoSets over the returned channel.
// The InoSets are ordered, by starting with the lowest inode and progressing
// through the highest (rather than returning InoSets in random order).
func (l LinkableInoSets) All() <-chan Set {
	// Make a slice of the Ino keys in LinkableInoSets, so that we can sort
	// them.  This allows us to output the full number of linkableInoSets
	// in a deterministic order (leading to more repeatable ordering of
	// link pairs across multiple dry-runs).  It's not completely
	// deterministic because there can still be multiple choices for
	// pre-linked src paths.
	i := 0
	sortedInos := make([]Ino, len(l))
	for ino := range l {
		sortedInos[i] = ino
		i++
	}
	sort.Slice(sortedInos, func(i, j int) bool { return sortedInos[i] < sortedInos[j] })

	out := make(chan Set)
	go func() {
		defer close(out)

		seen := NewSet()
		for _, startIno := range sortedInos {
			if seen.Has(startIno) {
				continue
			}
			out <- linkableInoSetHelper(l, startIno, seen)
		}
	}()
	return out
}
