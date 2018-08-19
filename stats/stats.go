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

import (
	np "hardlinkable/namepair"
	"os"
)

var Stats LinkingStats

func init() {
	Stats = NewLinkingStats()
}

type LinkDestinations struct {
	size      int64
	namepairs []np.Namepair
}

type LinkPair struct {
	Src np.Namepair
	Dst np.Namepair
}

type LinkingStats struct {
	numDirs             int64
	numFiles            int64
	numFilesTooSmall    int64
	numFilesTooLarge    int64
	numInodes           int64
	numComparisons      int64
	numEqualComparisons int64
	numMissedHashes     int64
	numFoundHashes      int64
	numInoSeqSearches   int64
	numInoSeqIterations int64
	numHashMismatches   int64

	linkPairs         []LinkPair
	existingHardlinks map[np.Namepair]LinkDestinations
}

type ExistingLink struct {
	SrcNamepair np.Namepair
	DstNamepair np.Namepair
	SrcFileinfo os.FileInfo
}

func NewLinkingStats() LinkingStats {
	ls := LinkingStats{
		existingHardlinks: make(map[np.Namepair]LinkDestinations),
	}
	return ls
}

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

func (s *LinkingStats) SearchedInoSeq() {
	s.numInoSeqSearches += 1
}

func (s *LinkingStats) IncInoSeqIterations() {
	s.numInoSeqIterations += 1
}

func (s *LinkingStats) NoHashMatch() {
	s.numHashMismatches += 1
}

func (s *LinkingStats) DidComparison() {
	s.numComparisons += 1
}

func (s *LinkingStats) FoundEqualFiles() {
	s.numEqualComparisons += 1
}

func (s *LinkingStats) FoundHardlinkableFiles(f1, f2 np.Namepair) {
	s.linkPairs = append(s.linkPairs, LinkPair{f1, f2})
}

func (s *LinkingStats) FoundExistingHardlink(existing ExistingLink) {
	srcNamepair := existing.SrcNamepair
	dstNamepair := existing.DstNamepair
	srcFileinfo := existing.SrcFileinfo
	linkDestinations, exists := s.existingHardlinks[srcNamepair]
	if !exists {
		size := srcFileinfo.Size()
		linkDestinations = LinkDestinations{size, make([]np.Namepair, 0)}
	}
	linkDestinations.namepairs = append(linkDestinations.namepairs, dstNamepair)
	s.existingHardlinks[srcNamepair] = linkDestinations
	//fmt.Println("currently linked: ", srcNamepair, linkDestinations)
}
