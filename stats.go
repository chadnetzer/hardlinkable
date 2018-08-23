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

var Stats LinkingStats

func init() {
	Stats = NewLinkingStats()
}

type LinkDestinations struct {
	size  uint64
	paths []Pathsplit
}

type LinkPair struct {
	Src Pathsplit
	Dst Pathsplit
}

type ExistingLink struct {
	LinkPair
	SrcStatinfo StatInfo
}

type CountingStats struct {
	numDirs               int64
	numFiles              int64
	numFilesTooSmall      int64
	numFilesTooLarge      int64
	numComparisons        int64
	numEqualComparisons   int64
	numInodes             int64
	numInodesConsolidated int64
	numMissedHashes       int64
	numPrevLinks          int64
	numNewLinks           int64
	numFoundHashes        int64
	numInoSeqSearches     int64
	numInoSeqIterations   int64
	numHashMismatches     int64
	numPrevBytesSaved     uint64
	numNewBytesSaved      uint64

}

type LinkingStats struct {
	CountingStats
	linkPairs     []LinkPair
	existingLinks map[Pathsplit]LinkDestinations
}

func NewLinkingStats() LinkingStats {
	ls := LinkingStats{
		existingLinks: make(map[Pathsplit]LinkDestinations),
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

func (s *LinkingStats) FoundNewLink(src, dst PathStat) {
	linkPair := LinkPair{src.Pathsplit, dst.Pathsplit}
	// Make optional to save space...
	s.linkPairs = append(s.linkPairs, linkPair)

	s.numNewLinks += 1
	if dst.Nlink == 1 {
		s.numNewBytesSaved += dst.Size
		s.numInodesConsolidated += 1
	}
}

func (s *LinkingStats) FoundExistingLink(e ExistingLink) {
	s.numPrevLinks += 1
	srcPath := e.Src
	dstPath := e.Dst
	srcStatinfo := e.SrcStatinfo
	linkDestinations, ok := s.existingLinks[srcPath]
	if !ok {
		size := srcStatinfo.Size
		linkDestinations = LinkDestinations{size, make([]Pathsplit, 0)}
	}
	linkDestinations.paths = append(linkDestinations.paths, dstPath)
	s.existingLinks[srcPath] = linkDestinations
	//fmt.Println("currently linked: ", srcPath, linkDestinations)
}

func humanize(n uint64) string {
	var s string
	var m string
	F := func(N uint64, div float64) string {
		reduced := float64(N) / div
		rounded := math.Round(reduced*1000) / 1000.0
		s = strconv.FormatFloat(rounded, 'f', -1, 64)
		return s
	}
	if n >= (uint64(1) << 50) {
		s = F(n, math.Pow(1024, 5))
		m = " PiB"
	} else if n >= (uint64(1) << 40) {
		s = F(n, math.Pow(1024, 4))
		m = " TiB"
	} else if n >= (uint64(1) << 30) {
		s = F(n, math.Pow(1024, 3))
		m = " GiB"
	} else if n >= (uint64(1) << 20) {
		s = F(n, math.Pow(1024, 2))
		m = " MiB"
	} else if n >= (uint64(1) << 10) {
		s = F(n, 1024.0)
		m = " KiB"
	} else {
		s = fmt.Sprintf("%d", n)
		m = " bytes"
	}

	return s + m
}
