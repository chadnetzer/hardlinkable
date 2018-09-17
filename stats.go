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

package hardlinkable

import (
	"fmt"
	P "hardlinkable/internal/pathpool"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type linkDestinations struct {
	size  uint64
	paths []P.Pathsplit
}

type linkPair struct {
	Src P.Pathsplit
	Dst P.Pathsplit
}

type CountingStats struct {
	DirCount               int64  `json:"dirCount"`
	FileCount              int64  `json:"fileCount"`
	FileTooSmallCount      int64  `json:"fileTooSmallCount"`
	FileTooLargeCount      int64  `json:"fileTooLargeCount"`
	ComparisonCount        int64  `json:"comparisonCount"`
	InodeCount             int64  `json:"inodeCount"`
	InodeRemovedCount      int64  `json:"inodeRemovedCount"`
	NlinkCount             int64  `json:"nlinkCount"`
	PrevLinkCount          int64  `json:"prevLinkCount"`
	NewLinkCount           int64  `json:"newLinkCount"`
	PrevLinkedByteAmount   uint64 `json:"prevLinkedByteAmount"`
	InodeRemovedByteAmount uint64 `json:"inodeRemovedByteAmount"`

	// Some stats on files that compared equal, but which had some
	// mismatching inode parameters.  This can be helpful for tuning the
	// command line options on subsequent runs.
	MismatchedMtimeCount int64  `json:"mismatchedMtimeCount"`
	MismatchedModeCount  int64  `json:"mismatchedModeCount"`
	MismatchedUidCount   int64  `json:"mismatchedUidCount"`
	MismatchedGidCount   int64  `json:"mismatchedGidCount"`
	MismatchedXattrCount int64  `json:"mismatchedXattrCount"`
	MismatchedTotalCount int64  `json:"mismatchedTotalCount"`
	MismatchedMtimeBytes uint64 `json:"mismatchedMtimeBytes"`
	MismatchedModeBytes  uint64 `json:"mismatchedModeBytes"`
	MismatchedUidBytes   uint64 `json:"mismatchedUidBytes"`
	MismatchedGidBytes   uint64 `json:"mismatchedGidBytes"`
	MismatchedXattrBytes uint64 `json:"mismatchedXattrBytes"`
	MismatchedTotalBytes uint64 `json:"mismatchedTotalBytes"`
	BytesCompared        uint64 `json:"bytesCompared"`

	// Debugging counts
	EqualComparisonCount int64 `json:"equalComparisonCount"`
	FoundHashCount       int64 `json:"foundHashCount"`
	MissedHashCount      int64 `json:"missedHashCount"`
	HashMismatchCount    int64 `json:"hashMismatchCount"`
	InoSeqSearchCount    int64 `json:"inoSeqSearchCount"`
	InoSeqIterationCount int64 `json:"inoSeqIterationCount"`
	DigestComputedCount  int64 `json:"digestComputedCount"`
}

type linkingStats struct {
	CountingStats
	StartTime     time.Time
	EndTime       time.Time
	LinkPairs     []linkPair
	ExistingLinks map[P.Pathsplit]linkDestinations
	Opts          Options
}

func newLinkingStats(o *Options) *linkingStats {
	ls := linkingStats{
		ExistingLinks: make(map[P.Pathsplit]linkDestinations),
		Opts:          *o,
	}
	return &ls
}

func (ls *linkingStats) FoundDirectory() {
	ls.DirCount += 1
}

func (ls *linkingStats) FoundFile() {
	ls.FileCount += 1
}

func (ls *linkingStats) FileAndDirectoryCount(fileCount, dirCount int64) {
	ls.FileCount = fileCount
	ls.DirCount = dirCount
}

func (ls *linkingStats) FoundFileTooSmall() {
	ls.FileTooSmallCount += 1
}

func (ls *linkingStats) FoundFileTooLarge() {
	ls.FileTooLargeCount += 1
}

func (ls *linkingStats) AddMismatchedMtimeBytes(size uint64) {
	ls.MismatchedMtimeCount += 1
	ls.MismatchedMtimeBytes += size
}

func (ls *linkingStats) AddMismatchedModeBytes(size uint64) {
	ls.MismatchedModeCount += 1
	ls.MismatchedModeBytes += size
}

func (ls *linkingStats) AddMismatchedUidBytes(size uint64) {
	ls.MismatchedUidCount += 1
	ls.MismatchedUidBytes += size
}

func (ls *linkingStats) AddMismatchedGidBytes(size uint64) {
	ls.MismatchedGidCount += 1
	ls.MismatchedGidBytes += size
}

func (ls *linkingStats) AddMismatchedXattrBytes(size uint64) {
	ls.MismatchedXattrCount += 1
	ls.MismatchedXattrBytes += size
}

func (ls *linkingStats) AddMismatchedTotalBytes(size uint64) {
	ls.MismatchedTotalCount += 1
	ls.MismatchedTotalBytes += size
}

func (ls *linkingStats) FoundInode(n uint32) {
	ls.InodeCount += 1
	ls.NlinkCount += int64(n)
}

func (ls *linkingStats) MissedHash() {
	ls.MissedHashCount += 1
}

func (ls *linkingStats) FoundHash() {
	ls.FoundHashCount += 1
}

func (ls *linkingStats) SearchedInoSeq() {
	ls.InoSeqSearchCount += 1
}

func (ls *linkingStats) IncInoSeqIterations() {
	ls.InoSeqIterationCount += 1
}

func (ls *linkingStats) NoHashMatch() {
	ls.HashMismatchCount += 1
}

func (ls *linkingStats) DidComparison() {
	ls.ComparisonCount += 1
}

func (ls *linkingStats) AddBytesCompared(n uint64) {
	ls.BytesCompared += n
}

func (ls *linkingStats) FoundEqualFiles() {
	ls.EqualComparisonCount += 1
}

func (ls *linkingStats) ComputedDigest() {
	ls.DigestComputedCount += 1
}

func (ls *linkingStats) FoundNewLink(src, dst P.Pathsplit) {
	if ls.Opts.newLinkStatsEnabled {
		lp := linkPair{src, dst}
		ls.LinkPairs = append(ls.LinkPairs, lp)
	}

	ls.NewLinkCount += 1
}

func (ls *linkingStats) FoundRemovedInode(size uint64) {
	ls.InodeRemovedByteAmount += size
	ls.InodeRemovedCount += 1
}

func (ls *linkingStats) FoundExistingLink(lp linkPair, size uint64) {
	ls.PrevLinkCount += 1
	ls.PrevLinkedByteAmount += size
	if !ls.Opts.existingLinkStatsEnabled {
		return
	}
	dests, ok := ls.ExistingLinks[lp.Src]
	if !ok {
		dests = linkDestinations{size, make([]P.Pathsplit, 0)}
	}
	dests.paths = append(dests.paths, lp.Dst)
	ls.ExistingLinks[lp.Src] = dests
}

func (ls *linkingStats) OutputResults() {
	if ls.Opts.existingLinkStatsEnabled {
		ls.OutputCurrentHardlinks()
		fmt.Println("")
	}
	if ls.Opts.newLinkStatsEnabled {
		ls.OutputLinkedPaths()
		if ls.Opts.StatsOutputEnabled {
			fmt.Println("")
		}
	}
	if ls.Opts.StatsOutputEnabled {
		ls.OutputLinkingStats()
	}
}

func (ls *linkingStats) OutputCurrentHardlinks() {
	s := make([]string, 0)
	s = append(s, "Currently hardlinked files")
	s = append(s, "--------------------------")
	for src, dsts := range ls.ExistingLinks {
		s = append(s, fmt.Sprintf("from: %v", src.Join()))
		for _, dst := range dsts.paths {
			s = append(s, fmt.Sprintf("  to: %v", dst.Join()))
		}
		totalSaved := dsts.size * uint64(len(dsts.paths)) // Can overflow
		s = append(s, fmt.Sprintf("Filesize: %v  Total saved: %v",
			humanize(dsts.size), humanize(totalSaved)))
	}
	fmt.Println(strings.Join(s, "\n"))
}

func (ls *linkingStats) OutputLinkedPaths() {
	s := make([]string, 0)
	if ls.Opts.LinkingEnabled {
		s = append(s, "Files that were hardlinked this run")
		s = append(s, "-----------------------------------")
	} else {
		s = append(s, "Files that are hardlinkable")
		s = append(s, "---------------------------")
	}
	prevPathsplit := P.Pathsplit{}
	for _, p := range ls.LinkPairs {
		if p.Src != prevPathsplit {
			s = append(s, "from: "+p.Src.Join())
			prevPathsplit = p.Src
		}
		s = append(s, "  to: "+p.Dst.Join())
	}
	fmt.Println(strings.Join(s, "\n"))
}

func (ls *linkingStats) OutputLinkingStats() {
	s := make([][]string, 0)
	s = statStr(s, "Hard linking statistics")
	s = statStr(s, "-----------------------")
	s = statStr(s, "Directories", ls.DirCount)
	s = statStr(s, "Files", ls.FileCount)
	if ls.Opts.LinkingEnabled {
		s = statStr(s, "Hardlinked this run", ls.NewLinkCount)
		s = statStr(s, "Removed inodes", ls.InodeRemovedCount)
	} else {
		s = statStr(s, "Hardlinkable this run", ls.NewLinkCount)
		s = statStr(s, "Removable inodes", ls.InodeRemovedCount)
	}
	s = statStr(s, "Currently linked bytes", ls.PrevLinkedByteAmount, humanizeParens(ls.PrevLinkedByteAmount))
	totalBytes := ls.PrevLinkedByteAmount + ls.InodeRemovedByteAmount
	var s1, s2 string
	if ls.Opts.LinkingEnabled {
		s1 = "Additional saved bytes"
		s2 = "Total saved bytes"
	} else {
		s1 = "Additional saveable bytes"
		s2 = "Total saveable bytes"
	}
	// Append some humanized size values to the byte string outputs
	s = statStr(s, s1, ls.InodeRemovedByteAmount, humanizeParens(ls.InodeRemovedByteAmount))
	s = statStr(s, s2, totalBytes, humanizeParens(totalBytes))

	duration := ls.EndTime.Sub(ls.StartTime)
	s = statStr(s, "Total run time", duration.Round(time.Millisecond).String())

	totalLinks := ls.PrevLinkCount + ls.NewLinkCount
	if ls.Opts.Verbosity > 0 || ls.Opts.DebugLevel > 0 {
		s = statStr(s, "Comparisons", ls.ComparisonCount)
		s = statStr(s, "Inodes", ls.InodeCount)
		unwalkedNlinks := ls.NlinkCount - ls.FileCount
		if unwalkedNlinks > 0 {
			unwalkedNlinkStr := fmt.Sprintf("(Unwalked Nlinks: %v)", unwalkedNlinks)
			s = statStr(s, "Inode total nlinks", ls.NlinkCount, unwalkedNlinkStr)
		}
		s = statStr(s, "Existing links", ls.PrevLinkCount)
		s = statStr(s, "Total old + new links", totalLinks)
		if ls.FileTooLargeCount > 0 {
			s = statStr(s, "Total too large files", ls.FileTooLargeCount)
		}
		if ls.FileTooSmallCount > 0 {
			s = statStr(s, "Total too small files", ls.FileTooSmallCount)
		}
		if ls.MismatchedMtimeCount > 0 {
			s = statStr(s, "Equal files w/ unequal time", ls.MismatchedMtimeCount,
				humanizeParens(ls.MismatchedMtimeBytes))
		}
		if ls.MismatchedModeCount > 0 {
			s = statStr(s, "Equal files w/ unequal mode", ls.MismatchedModeCount,
				humanizeParens(ls.MismatchedModeBytes))
		}
		if ls.MismatchedUidCount > 0 {
			s = statStr(s, "Equal files w/ unequal uid", ls.MismatchedUidCount,
				humanizeParens(ls.MismatchedUidBytes))
		}
		if ls.MismatchedGidCount > 0 {
			s = statStr(s, "Equal files w/ unequal gid", ls.MismatchedGidCount,
				humanizeParens(ls.MismatchedGidBytes))
		}
		if ls.MismatchedXattrCount > 0 {
			s = statStr(s, "Equal files w/ unequal xattr", ls.MismatchedXattrCount,
				humanizeParens(ls.MismatchedXattrBytes))
		}
		if ls.MismatchedTotalBytes > 0 {
			s = statStr(s, "Total equal file mismatches", ls.MismatchedTotalCount,
				humanizeParens(ls.MismatchedTotalBytes))
		}
		if ls.BytesCompared > 0 {
			s = statStr(s, "Total bytes compared", ls.BytesCompared,
				humanizeParens(ls.BytesCompared))
		}

		remainingInodes := ls.InodeCount - ls.InodeRemovedCount
		s = statStr(s, "Total remaining inodes", remainingInodes)
	}
	if ls.Opts.DebugLevel > 0 {
		// add additional stat output onto the last string
		s = statStr(s, "Total file hash hits", ls.FoundHashCount,
			fmt.Sprintf("misses: %v  sum total: %v", ls.MissedHashCount,
				ls.FoundHashCount+ls.MissedHashCount))
		s = statStr(s, "Total hash mismatches", ls.HashMismatchCount,
			fmt.Sprintf("(+ total links: %v)", ls.HashMismatchCount+totalLinks))
		s = statStr(s, "Total hash searches", ls.InoSeqSearchCount)
		avgItersPerSearch := "N/A"
		if ls.InoSeqIterationCount > 0 {
			avg := float64(ls.InoSeqIterationCount) / float64(ls.InoSeqSearchCount)
			avgItersPerSearch = fmt.Sprintf("%.1f", avg)
		}
		s = statStr(s, "Total hash list iterations", ls.InoSeqIterationCount,
			fmt.Sprintf("(avg per search: %v)", avgItersPerSearch))
		s = statStr(s, "Total equal comparisons", ls.EqualComparisonCount)
		s = statStr(s, "Total digests computed", ls.DigestComputedCount)
	}

	if ls.Opts.DebugLevel > 1 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		s = statStr(s, "Mem Alloc", humanize(m.Alloc))
		s = statStr(s, "Mem Sys", humanize(m.Sys))
		s = statStr(s, "Num live objects", m.Mallocs-m.Frees)
	}
	printSlices(s)
}

// Add a new row of string colums to the given slice of string slices
func statStr(a [][]string, args ...interface{}) [][]string {
	s := make([]string, 0)
	for _, arg := range args {
		s = append(s, fmt.Sprintf("%v", arg))
	}
	return append(a, s)
}

// Columnate printing of a slice of string slices (ie. a list of string
// columns)
func printSlices(a [][]string) {
	numCols := 0
	for _, c := range a {
		if len(c) > numCols {
			numCols = len(c)
		}
	}
	colWidths := make([]int, numCols)
	for _, c := range a {
		for i, s := range c {
			if len(s) > colWidths[i] {
				colWidths[i] = len(s)
			}
		}
	}
	for _, c := range a {
		for i, s := range c {
			if i == 1 {
				fmt.Print(" :")
			}
			if i >= 1 {
				fmt.Print(" ")
			}
			if i >= 2 {
				fmt.Print(" ")
			}
			fmtStr := "%-" + fmt.Sprintf("%v", colWidths[i]) + "s"
			fmt.Printf(fmtStr, s)
		}
		fmt.Println()
	}
}

// Return a string with bytecount "humanized" to a shortened amount
func humanize(n uint64) string {
	// -1 precision removes trailing zeros
	return humanizeWithPrecision(n, -1)
}

// humanizeWithPrecision allows providing FormatFloat precision value
func humanizeWithPrecision(n uint64, prec int) string {
	var s string
	var m string
	F := func(N uint64, div float64) string {
		reduced := float64(N) / div
		rounded := math.Round(reduced*1000) / 1000.0
		s = strconv.FormatFloat(rounded, 'f', prec, 64)
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

// Return the humanized number count as a string surrounded by parens
func humanizeParens(n uint64) string {
	return fmt.Sprintf("(%v)", humanize(n))
}
