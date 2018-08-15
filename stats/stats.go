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

package stats

type LinkingStats struct {
	numDirs          int
	numFiles         int
	numFilesTooSmall int
	numFilesTooLarge int
	numInodes        int
	numMissedHashes  int
	numFoundHashes   int
}

var Stats LinkingStats

func (s *LinkingStats) FoundDirectory() {
	s.numDirs += 1
}

func (s *LinkingStats) FoundFile() {
	s.numFiles += 1
}

func (s *LinkingStats) FoundFileTooSmall() {
	s.numFilesTooSmall += 1
}

func (s *LinkingStats) FoundFileTooLarge() {
	s.numFilesTooLarge += 1
}

func (s *LinkingStats) FoundInode() {
	s.numInodes += 1
}

func (s *LinkingStats) MissedHash() {
	s.numMissedHashes += 1
}

func (s *LinkingStats) FoundHash() {
	s.numFoundHashes += 1
}
