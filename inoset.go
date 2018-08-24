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

type Ino = uint64

type InoSet map[Ino]struct{}

var exists = struct{}{}

func NewInoSet(inos ...Ino) InoSet {
	set := make(map[Ino]struct{})
	for _, ino := range inos {
		set[ino] = exists
	}
	return set
}

func (s InoSet) Add(ino Ino) {
	s[ino] = exists
}

// Return true if given Ino is in the InoSet
func (s InoSet) Has(ino Ino) bool {
	_, ok := s[ino]
	return ok
}

func (s InoSet) Copy() InoSet {
	newSet := NewInoSet()
	for k := range s {
		newSet[k] = exists
	}
	return newSet
}

func (s InoSet) Intersection(set2 InoSet) InoSet {
	resultSet := NewInoSet()
	var little, big *InoSet
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

func (s InoSet) Difference(other InoSet) InoSet {
	// Iterate over smaller set
	resultSet := NewInoSet()
	for k := range s {
		if _, ok := other[k]; !ok {
			resultSet.Add(k)
		}
	}
	return resultSet
}
