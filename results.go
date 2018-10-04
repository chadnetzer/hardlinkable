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
	"encoding/json"
	"fmt"
	P "hardlinkable/internal/pathpool"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type RunPhases int

const (
	StartPhase RunPhases = iota // nothing yet happened
	WalkPhase
	LinkPhase
	EndPhase
)

// RunStats holds information about counts, the number of files found to be
// linkable, the bytes that linking would save (or did save), and a variety of
// related, useful, or just interesting information gathered during the Run().
type RunStats struct {
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

// Results contains the RunStats information, as well as the found existing and
// new links.  It also includes a measurement of how long the Run() took to
// execute, and the Options that were used to perform the Run().
type Results struct {
	// Link member strings are pathnames
	ExistingLinks     map[string][]string `json:"existingLinks"`
	ExistingLinkSizes map[string]uint64   `json:"existingLinkSizes"`
	LinkPaths         [][]string          `json:"linkPaths"`
	RunStats
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	RunTime   string    `json:"runTime"`
	Opts      Options   `json:"options"`

	// Set to true when Run() has completed successfully
	RunSuccessful bool `json:"runSuccessful"`

	// Record which 'phase' we've gotten to in the algorithms, in case of
	// early termination of the run.
	Phase RunPhases
}

func newResults(o *Options) *Results {
	r := Results{
		ExistingLinks:     make(map[string][]string),
		ExistingLinkSizes: make(map[string]uint64),
		Opts:              *o,
	}
	return &r
}

// foundFile keeps a running count of the files found (not counting those that
// are excluded).  The final tally can be overwritten when all paths are
// walked, but the running tally is used by the progress interfaces while the
// walk is occurring.
func (r *Results) foundFile() {
	r.FileCount += 1
}

func (r *Results) fileAndDirectoryCount(fileCount, dirCount int64) {
	r.FileCount = fileCount
	r.DirCount = dirCount
}

func (r *Results) foundFileTooSmall() {
	r.FileTooSmallCount += 1
}

func (r *Results) foundFileTooLarge() {
	r.FileTooLargeCount += 1
}

func (r *Results) addMismatchedMtimeBytes(size uint64) {
	r.MismatchedMtimeCount += 1
	r.MismatchedMtimeBytes += size
}

func (r *Results) addMismatchedModeBytes(size uint64) {
	r.MismatchedModeCount += 1
	r.MismatchedModeBytes += size
}

func (r *Results) addMismatchedUidBytes(size uint64) {
	r.MismatchedUidCount += 1
	r.MismatchedUidBytes += size
}

func (r *Results) addMismatchedGidBytes(size uint64) {
	r.MismatchedGidCount += 1
	r.MismatchedGidBytes += size
}

func (r *Results) addMismatchedXattrBytes(size uint64) {
	r.MismatchedXattrCount += 1
	r.MismatchedXattrBytes += size
}

func (r *Results) addMismatchedTotalBytes(size uint64) {
	r.MismatchedTotalCount += 1
	r.MismatchedTotalBytes += size
}

func (r *Results) foundInode(n uint32) {
	r.InodeCount += 1
	r.NlinkCount += int64(n)
}

func (r *Results) foundRemovedInode(size uint64) {
	r.InodeRemovedByteAmount += size
	r.InodeRemovedCount += 1
}

func (r *Results) missedHash() {
	r.MissedHashCount += 1
}

func (r *Results) foundHash() {
	r.FoundHashCount += 1
}

func (r *Results) searchedInoSeq() {
	r.InoSeqSearchCount += 1
}

func (r *Results) incInoSeqIterations() {
	r.InoSeqIterationCount += 1
}

func (r *Results) noHashMatch() {
	r.HashMismatchCount += 1
}

func (r *Results) didComparison() {
	r.ComparisonCount += 1
}

func (r *Results) addBytesCompared(n uint64) {
	r.BytesCompared += n
}

func (r *Results) foundEqualFiles() {
	r.EqualComparisonCount += 1
}

func (r *Results) computedDigest() {
	r.DigestComputedCount += 1
}

func (r *Results) start() {
	r.StartTime = time.Now()
}

func (r *Results) end() {
	r.EndTime = time.Now()
	duration := r.EndTime.Sub(r.StartTime)
	r.RunTime = duration.Round(time.Millisecond).String()
}

func (r *Results) runCompletedSuccessfully() {
	r.Phase = EndPhase
	r.RunSuccessful = true
}

// Track the count of new links, and optionally keep a list of linkable or
// linked pathnames for later output.
func (r *Results) foundNewLink(srcP, dstP P.Pathsplit) {
	r.NewLinkCount += 1
	if !r.Opts.StoreNewLinkResults {
		return
	}
	src := srcP.Join()
	dst := dstP.Join()
	N := len(r.LinkPaths)
	if N == 0 {
		r.LinkPaths = [][]string{[]string{src, dst}}
	} else {
		prevSrc := r.LinkPaths[N-1][0]
		if src == prevSrc {
			r.LinkPaths[N-1] = append(r.LinkPaths[N-1], dst)
		} else {
			pair := []string{src, dst}
			r.LinkPaths = append(r.LinkPaths, pair)
		}
	}
}

// Track count of existing links found during walk, and optionally keep a list
// of them and their sizes for later output.
func (r *Results) foundExistingLink(srcP P.Pathsplit, dstP P.Pathsplit, size uint64) {
	r.PrevLinkCount += 1
	r.PrevLinkedByteAmount += size
	if !r.Opts.StoreExistingLinkResults {
		return
	}
	src := srcP.Join()
	dst := dstP.Join()
	dests, ok := r.ExistingLinks[src]
	if !ok {
		dests = []string{dst}
		r.ExistingLinkSizes[src] = size
	} else {
		dests = append(dests, dst)
	}
	r.ExistingLinks[src] = dests

	panicIf(size != r.ExistingLinkSizes[src],
		fmt.Sprintf("Existing link %v size %v, expected size %v",
			src, size, r.ExistingLinkSizes[src]))
}

// OutputResults prints results in text form, including existing links that
// were found, new pathnames that were discovered to be linkable, and stats
// about the run giving information on the amount of data that can be saved (or
// was saved if linking was enabled).
func (r *Results) OutputResults() {
	r.OutputExistingLinks()
	if len(r.ExistingLinks) > 0 && (len(r.LinkPaths) > 0 || r.Opts.ShowRunStats) {
		fmt.Println("")
	}
	r.OutputNewLinks()
	if len(r.LinkPaths) > 0 && r.Opts.ShowRunStats {
		fmt.Println("")
	}
	if r.Opts.ShowRunStats || r.Opts.ShowExtendedRunStats {
		r.OutputRunStats()
	}
}

// OutputExistingLinks shows in text form the existing links that were found by
// Run.
func (r *Results) OutputExistingLinks() {
	if len(r.ExistingLinks) == 0 {
		return
	}
	s := make([]string, 0)
	s = append(s, "Currently hardlinked files")
	s = append(s, "--------------------------")
	for src, dsts := range r.ExistingLinks {
		s = append(s, fmt.Sprintf("from: %v", src))
		for _, dst := range dsts {
			s = append(s, fmt.Sprintf("  to: %v", dst))
		}
		size := r.ExistingLinkSizes[src]
		totalSaved := size * uint64(len(dsts)) // Can overflow
		s = append(s, fmt.Sprintf("Filesize: %v  Total saved: %v",
			humanize(size), humanize(totalSaved)))
	}
	fmt.Println(strings.Join(s, "\n"))
}

// OutputNewLinks shows in text form the pathnames that were discovered to be
// linkable.
func (r *Results) OutputNewLinks() {
	if len(r.LinkPaths) == 0 {
		return
	}
	s := make([]string, 0)
	if r.Opts.LinkingEnabled {
		s = append(s, "Files that were hardlinked this run")
		s = append(s, "-----------------------------------")
	} else {
		s = append(s, "Files that are hardlinkable")
		s = append(s, "---------------------------")
	}
	for _, paths := range r.LinkPaths {
		for i, path := range paths {
			if i == 0 {
				s = append(s, "from: "+path)
			} else {
				s = append(s, "  to: "+path)
			}
		}
	}
	fmt.Println(strings.Join(s, "\n"))
}

// OutputRunStats show information about how many files could be linked, how
// much space would be saved, and other information on inodes, comparisons,
// etc.  If linking was enabled, it displays the information on links that were
// actually made and space actually saved (which should equal the predicted
// amounts).
func (r *Results) OutputRunStats() {
	s := make([][]string, 0)
	s = statStr(s, "Hard linking statistics")
	s = statStr(s, "-----------------------")
	s = statStr(s, "Directories", r.DirCount)
	s = statStr(s, "Files", r.FileCount)
	if r.Opts.LinkingEnabled {
		s = statStr(s, "Hardlinked this run", r.NewLinkCount)
		s = statStr(s, "Removed inodes", r.InodeRemovedCount)
	} else {
		s = statStr(s, "Hardlinkable this run", r.NewLinkCount)
		s = statStr(s, "Removable inodes", r.InodeRemovedCount)
	}
	s = statStr(s, "Currently linked bytes", r.PrevLinkedByteAmount, humanizeParens(r.PrevLinkedByteAmount))
	totalBytes := r.PrevLinkedByteAmount + r.InodeRemovedByteAmount
	var s1, s2 string
	if r.Opts.LinkingEnabled {
		s1 = "Additional saved bytes"
		s2 = "Total saved bytes"
	} else {
		s1 = "Additional saveable bytes"
		s2 = "Total saveable bytes"
	}
	// Append some humanized size values to the byte string outputs
	s = statStr(s, s1, r.InodeRemovedByteAmount, humanizeParens(r.InodeRemovedByteAmount))
	s = statStr(s, s2, totalBytes, humanizeParens(totalBytes))

	s = statStr(s, "Total run time", r.RunTime)

	totalLinks := r.PrevLinkCount + r.NewLinkCount
	if r.Opts.ShowExtendedRunStats || r.Opts.DebugLevel > 0 {
		s = statStr(s, "Comparisons", r.ComparisonCount)
		s = statStr(s, "Inodes", r.InodeCount)
		unwalkedNlinks := r.NlinkCount - r.FileCount
		if unwalkedNlinks > 0 {
			unwalkedNlinkStr := fmt.Sprintf("(Unwalked Nlinks: %v)", unwalkedNlinks)
			s = statStr(s, "Inode total nlinks", r.NlinkCount, unwalkedNlinkStr)
		}
		s = statStr(s, "Existing links", r.PrevLinkCount)
		s = statStr(s, "Total old + new links", totalLinks)
		if r.FileTooLargeCount > 0 {
			s = statStr(s, "Total too large files", r.FileTooLargeCount)
		}
		if r.FileTooSmallCount > 0 {
			s = statStr(s, "Total too small files", r.FileTooSmallCount)
		}
		if r.MismatchedMtimeCount > 0 {
			s = statStr(s, "Equal files w/ unequal time", r.MismatchedMtimeCount,
				humanizeParens(r.MismatchedMtimeBytes))
		}
		if r.MismatchedModeCount > 0 {
			s = statStr(s, "Equal files w/ unequal mode", r.MismatchedModeCount,
				humanizeParens(r.MismatchedModeBytes))
		}
		if r.MismatchedUidCount > 0 {
			s = statStr(s, "Equal files w/ unequal uid", r.MismatchedUidCount,
				humanizeParens(r.MismatchedUidBytes))
		}
		if r.MismatchedGidCount > 0 {
			s = statStr(s, "Equal files w/ unequal gid", r.MismatchedGidCount,
				humanizeParens(r.MismatchedGidBytes))
		}
		if r.MismatchedXattrCount > 0 {
			s = statStr(s, "Equal files w/ unequal xattr", r.MismatchedXattrCount,
				humanizeParens(r.MismatchedXattrBytes))
		}
		if r.MismatchedTotalBytes > 0 {
			s = statStr(s, "Total equal file mismatches", r.MismatchedTotalCount,
				humanizeParens(r.MismatchedTotalBytes))
		}
		if r.BytesCompared > 0 {
			s = statStr(s, "Total bytes compared", r.BytesCompared,
				humanizeParens(r.BytesCompared))
		}

		remainingInodes := r.InodeCount - r.InodeRemovedCount
		s = statStr(s, "Total remaining inodes", remainingInodes)
	}

	if r.Opts.DebugLevel > 0 {
		// add additional stat output onto the last string
		s = statStr(s, "Total file hash hits", r.FoundHashCount,
			fmt.Sprintf("misses: %v  sum total: %v", r.MissedHashCount,
				r.FoundHashCount+r.MissedHashCount))
		s = statStr(s, "Total hash mismatches", r.HashMismatchCount,
			fmt.Sprintf("(+ total links: %v)", r.HashMismatchCount+totalLinks))
		s = statStr(s, "Total hash searches", r.InoSeqSearchCount)
		avgItersPerSearch := "N/A"
		if r.InoSeqIterationCount > 0 {
			avg := float64(r.InoSeqIterationCount) / float64(r.InoSeqSearchCount)
			avgItersPerSearch = fmt.Sprintf("%.1f", avg)
		}
		s = statStr(s, "Total hash list iterations", r.InoSeqIterationCount,
			fmt.Sprintf("(avg per search: %v)", avgItersPerSearch))
		s = statStr(s, "Total equal comparisons", r.EqualComparisonCount)
		s = statStr(s, "Total digests computed", r.DigestComputedCount)
	}

	if r.Opts.DebugLevel > 1 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		s = statStr(s, "Mem Alloc", humanize(m.Alloc))
		s = statStr(s, "Mem Sys", humanize(m.Sys))
		s = statStr(s, "Num live objects", m.Mallocs-m.Frees)
	}
	printSlices(s)
}

// OutputJSONResults outputs a JSON formatted object with all the information
// gathered by Run() about existing and new links, and stats on space saved,
// etc.
func (r *Results) OutputJSONResults() {
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
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
