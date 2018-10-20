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
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	P "github.com/chadnetzer/hardlinkable/internal/pathpool"
)

// RunPhases is an enum that indicates which phase of the Run() algorithm is
// being executed.
type RunPhases int

const (
	// StartPhase indicates the Run() algorithm hasn't started
	StartPhase RunPhases = iota
	// WalkPhase indicates the directory/file walk which gathers info
	WalkPhase
	// LinkPhase indicates that the pathname link pairs are being computed
	LinkPhase
	// EndPhase indicates the Run() has finished
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
	ExistingLinkCount      int64  `json:"existingLinkCount"`
	NewLinkCount           int64  `json:"newLinkCount"`
	ExistingLinkByteAmount uint64 `json:"existingLinkByteAmount"`
	InodeRemovedByteAmount uint64 `json:"inodeRemovedByteAmount"`
	BytesCompared          uint64 `json:"bytesCompared"`

	// Some stats on files that compared equal, but which had some
	// mismatching inode parameters.  This can be helpful for tuning the
	// command line options on subsequent runs.
	MismatchedMtimeCount int64  `json:"mismatchedMtimeCount"`
	MismatchedModeCount  int64  `json:"mismatchedModeCount"`
	MismatchedUIDCount   int64  `json:"mismatchedUIDCount"`
	MismatchedGIDCount   int64  `json:"mismatchedGIDCount"`
	MismatchedXAttrCount int64  `json:"mismatchedXAttrCount"`
	MismatchedTotalCount int64  `json:"mismatchedTotalCount"`
	MismatchedMtimeBytes uint64 `json:"mismatchedMtimeBytes"`
	MismatchedModeBytes  uint64 `json:"mismatchedModeBytes"`
	MismatchedUIDBytes   uint64 `json:"mismatchedUIDBytes"`
	MismatchedGIDBytes   uint64 `json:"mismatchedGIDBytes"`
	MismatchedXAttrBytes uint64 `json:"mismatchedXAttrBytes"`
	MismatchedTotalBytes uint64 `json:"mismatchedTotalBytes"`

	// Counts of file I/O errors (reading, linking, etc.)
	SkippedDirErrCount  int64 `json:"skippedDirErrCount"`
	SkippedFileErrCount int64 `json:"skippedFileErrCount"`
	SkippedLinkErrCount int64 `json:"skippedLinkErrCount"`

	// Counts of files and dirs excluded by the Regex matches
	ExcludedDirCount  int64 `json:"excludedDirCount"`
	ExcludedFileCount int64 `json:"excludedFileCount"`
	IncludedFileCount int64 `json:"includedFileCount"`

	// Count of how many setuid and setgid files were encountered (and skipped)
	SkippedSetuidCount int64 `json:"skippedSetuidCount"`
	SkippedSetgidCount int64 `json:"skippedSetgidCount"`

	// Also keep track of files with bits other than the permission bits
	// set (other than setuid/setgid and bits already excluded by "regular"
	// file bits)
	SkippedNonPermBitCount int64 `json:"skippedNonPermBitCount"`

	// Debugging counts
	EqualComparisonCount int64 `json:"equalComparisonCount"`
	FoundHashCount       int64 `json:"foundHashCount"`
	MissedHashCount      int64 `json:"missedHashCount"`
	HashMismatchCount    int64 `json:"hashMismatchCount"`
	InoSeqSearchCount    int64 `json:"inoSeqSearchCount"`
	InoSeqIterationCount int64 `json:"inoSeqIterationCount"`
	DigestComputedCount  int64 `json:"digestComputedCount"`

	// Counts of how many times the hardlinkFiles() func wasn't able to
	// successfully change inode times and/or uid/gid.  Since we ignore
	// such errors and continue anyway (ie. it's a best-effort attempt,
	// rather than a guarantee), the counts are debugging info.
	FailedLinkChtimesCount int64 `json:"failedLinkChtimesCount"`
	FailedLinkChownCount   int64 `json:"failedLinkChownCount"`
}

// Results contains the RunStats information, as well as the found existing and
// new links.  It also includes a measurement of how long the Run() took to
// execute, and the Options that were used to perform the Run().
type Results struct {
	// Link member strings are pathnames
	ExistingLinks     map[string][]string `json:"existingLinks"`
	ExistingLinkSizes map[string]uint64   `json:"existingLinkSizes"`
	LinkPaths         [][]string          `json:"linkPaths"`
	SkippedLinkPaths  [][]string          `json:"skippedLinkPaths"` // Skipped when link failed
	RunStats
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	RunTime   string    `json:"runTime"`
	Opts      Options   `json:"options"`

	// Set to true when Run() has completed successfully
	RunSuccessful bool `json:"runSuccessful"`

	// Record which 'phase' we've gotten to in the algorithms, in case of
	// early termination of the run.
	Phase RunPhases `json:"phase"`
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
	r.FileCount++
}

func (r *Results) foundFileTooSmall() {
	r.FileTooSmallCount++
}

func (r *Results) foundFileTooLarge() {
	r.FileTooLargeCount++
}

func (r *Results) addMismatchedMtimeBytes(size uint64) {
	r.MismatchedMtimeCount++
	r.MismatchedMtimeBytes += size
}

func (r *Results) addMismatchedModeBytes(size uint64) {
	r.MismatchedModeCount++
	r.MismatchedModeBytes += size
}

func (r *Results) addMismatchedUIDBytes(size uint64) {
	r.MismatchedUIDCount++
	r.MismatchedUIDBytes += size
}

func (r *Results) addMismatchedGIDBytes(size uint64) {
	r.MismatchedGIDCount++
	r.MismatchedGIDBytes += size
}

func (r *Results) addMismatchedXAttrBytes(size uint64) {
	r.MismatchedXAttrCount++
	r.MismatchedXAttrBytes += size
}

func (r *Results) addMismatchedTotalBytes(size uint64) {
	r.MismatchedTotalCount++
	r.MismatchedTotalBytes += size
}

func (r *Results) foundInode(n uint64) {
	r.InodeCount++
	r.NlinkCount += int64(n)
}

func (r *Results) foundRemovedInode(size uint64) {
	r.InodeRemovedCount++
	r.InodeRemovedByteAmount += size
}

func (r *Results) foundSetuidFile() {
	r.SkippedSetuidCount++
}

func (r *Results) foundSetgidFile() {
	r.SkippedSetgidCount++
}

func (r *Results) foundNonPermBitFile() {
	r.SkippedNonPermBitCount++
}

func (r *Results) missedHash() {
	r.MissedHashCount++
}

func (r *Results) foundHash() {
	r.FoundHashCount++
}

func (r *Results) searchedInoSeq() {
	r.InoSeqSearchCount++
}

func (r *Results) incInoSeqIterations() {
	r.InoSeqIterationCount++
}

func (r *Results) noHashMatch() {
	r.HashMismatchCount++
}

func (r *Results) didComparison() {
	r.ComparisonCount++
}

func (r *Results) addBytesCompared(n uint64) {
	r.BytesCompared += n
}

func (r *Results) foundEqualFiles() {
	r.EqualComparisonCount++
}

func (r *Results) computedDigest() {
	r.DigestComputedCount++
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
	r.NewLinkCount++
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
	r.ExistingLinkCount++
	r.ExistingLinkByteAmount += size
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

// Track the count of skipped new links (ie. those where linking was attempted,
// but failed), and optionally keep a list of linkable or linked pathnames for
// later output.
func (r *Results) skippedNewLink(srcP, dstP P.Pathsplit) {
	r.SkippedLinkErrCount++
	if !r.Opts.StoreNewLinkResults {
		return
	}
	src := srcP.Join()
	dst := dstP.Join()
	N := len(r.SkippedLinkPaths)
	if N == 0 {
		r.SkippedLinkPaths = [][]string{[]string{src, dst}}
	} else {
		prevSrc := r.SkippedLinkPaths[N-1][0]
		if src == prevSrc {
			r.SkippedLinkPaths[N-1] = append(r.SkippedLinkPaths[N-1], dst)
		} else {
			pair := []string{src, dst}
			r.SkippedLinkPaths = append(r.SkippedLinkPaths, pair)
		}
	}
}

// OutputResults prints results in text form, including existing links that
// were found, new pathnames that were discovered to be linkable, and stats
// about the run giving information on the amount of data that can be saved (or
// was saved if linking was enabled).
func (r *Results) OutputResults() {
	showStats := r.Opts.ShowRunStats || r.Opts.ShowExtendedRunStats

	r.OutputExistingLinks()
	if len(r.ExistingLinks) > 0 &&
		(len(r.LinkPaths) > 0 || len(r.SkippedLinkPaths) > 0 || showStats) {
		fmt.Println("")
	}

	r.OutputNewLinks()
	if len(r.LinkPaths) > 0 && (len(r.SkippedLinkPaths) > 0 || showStats) {
		fmt.Println("")
	}

	r.OutputSkippedNewLinks()
	if len(r.SkippedLinkPaths) > 0 && showStats {
		fmt.Println("")
	}

	if showStats {
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
			Humanize(size), Humanize(totalSaved)))
		fmt.Println(strings.Join(s, "\n"))
		s = []string{}
	}
	if len(s) > 0 {
		fmt.Println(strings.Join(s, "\n"))
	}
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
	outputLinkPaths(s, r.LinkPaths)
}

// OutputSkippedNewLinks shows in text form the pathnames that were skipped due
// to linking errors.
func (r *Results) OutputSkippedNewLinks() {
	if len(r.SkippedLinkPaths) == 0 {
		return
	}
	s := make([]string, 0)
	s = append(s, "Files that had linking errors this run")
	s = append(s, "--------------------------------------")
	outputLinkPaths(s, r.SkippedLinkPaths)
	fmt.Println(strings.Join(s, "\n"))
}

// outputLinkPaths is a helper for outputting LinkPaths slices
func outputLinkPaths(s []string, lp [][]string) {
	for _, paths := range lp {
		for i, path := range paths {
			if i == 0 {
				s = append(s, "from: "+path)
			} else {
				s = append(s, "  to: "+path)
			}
		}
		fmt.Println(strings.Join(s, "\n"))
		s = []string{}
	}
	if len(s) > 0 {
		fmt.Println(strings.Join(s, "\n"))
	}
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
	if !r.RunSuccessful {
		var phase string
		switch r.Phase {
		case StartPhase:
			phase = "Start"
		case WalkPhase:
			phase = "File walk"
		case LinkPhase:
			phase = "Linking"
		default:
			phase = "End"
		}
		s = statStr(s, "Run stopped early in phase", phase)
	}
	s = statStr(s, "Directories", r.DirCount)
	s = statStr(s, "Files", r.FileCount)
	if r.Opts.LinkingEnabled {
		s = statStr(s, "Hardlinked this run", r.NewLinkCount)
		s = statStr(s, "Removed inodes", r.InodeRemovedCount)
	} else {
		s = statStr(s, "Hardlinkable this run", r.NewLinkCount)
		s = statStr(s, "Removable inodes", r.InodeRemovedCount)
	}
	s = statStr(s, "Currently linked bytes", r.ExistingLinkByteAmount, humanizeParens(r.ExistingLinkByteAmount))
	totalBytes := r.ExistingLinkByteAmount + r.InodeRemovedByteAmount
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

	totalLinks := r.ExistingLinkCount + r.NewLinkCount
	if r.Opts.ShowExtendedRunStats || r.Opts.DebugLevel > 0 {
		s = statStr(s, "Comparisons", r.ComparisonCount)
		s = statStr(s, "Inodes", r.InodeCount)
		unwalkedNlinks := r.NlinkCount - r.FileCount
		if unwalkedNlinks > 0 {
			unwalkedNlinkStr := fmt.Sprintf("(Unwalked Nlinks: %v)", unwalkedNlinks)
			s = statStr(s, "Inode total nlinks", r.NlinkCount, unwalkedNlinkStr)
		}
		s = statStr(s, "Existing links", r.ExistingLinkCount)
		s = statStr(s, "Total old + new links", totalLinks)
		if r.FileTooLargeCount > 0 {
			s = statStr(s, "Total too large files", r.FileTooLargeCount)
		}
		if r.FileTooSmallCount > 0 {
			s = statStr(s, "Total too small files", r.FileTooSmallCount)
		}
		if r.ExcludedDirCount > 0 {
			s = statStr(s, "Total excluded dirs", r.ExcludedDirCount)
		}
		if r.ExcludedFileCount > 0 {
			s = statStr(s, "Total excluded files", r.ExcludedFileCount)
		}
		if r.IncludedFileCount > 0 {
			s = statStr(s, "Total included files", r.IncludedFileCount)
		}
		if r.MismatchedMtimeCount > 0 {
			s = statStr(s, "Equal files w/ unequal time", r.MismatchedMtimeCount,
				humanizeParens(r.MismatchedMtimeBytes))
		}
		if r.MismatchedModeCount > 0 {
			s = statStr(s, "Equal files w/ unequal mode", r.MismatchedModeCount,
				humanizeParens(r.MismatchedModeBytes))
		}
		if r.MismatchedUIDCount > 0 {
			s = statStr(s, "Equal files w/ unequal uid", r.MismatchedUIDCount,
				humanizeParens(r.MismatchedUIDBytes))
		}
		if r.MismatchedGIDCount > 0 {
			s = statStr(s, "Equal files w/ unequal gid", r.MismatchedGIDCount,
				humanizeParens(r.MismatchedGIDBytes))
		}
		if r.MismatchedXAttrCount > 0 {
			s = statStr(s, "Equal files w/ unequal xattr", r.MismatchedXAttrCount,
				humanizeParens(r.MismatchedXAttrBytes))
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

		if r.SkippedSetuidCount > 0 {
			s = statStr(s, "Skipped setuid files", r.SkippedSetuidCount)
		}
		if r.SkippedSetgidCount > 0 {
			s = statStr(s, "Skipped setgid files", r.SkippedSetgidCount)
		}
		if r.SkippedNonPermBitCount > 0 {
			s = statStr(s, "Skipped files with non-perm bits set", r.SkippedNonPermBitCount)
		}
		if r.SkippedDirErrCount > 0 {
			s = statStr(s, "Dir errors this run", r.SkippedDirErrCount)
		}
		if r.SkippedFileErrCount > 0 {
			s = statStr(s, "File errors this run", r.SkippedFileErrCount)
		}
		if r.SkippedLinkErrCount > 0 {
			s = statStr(s, "Link errors this run", r.SkippedLinkErrCount)
		}
	}

	if r.Opts.DebugLevel > 0 {
		// add additional stat output onto the last string
		s = statStr(s, "Total file hash hits", r.FoundHashCount,
			fmt.Sprintf("misses: %v  sum total: %v", r.MissedHashCount,
				r.FoundHashCount+r.MissedHashCount))
		s = statStr(s, "Total hash mismatches", r.HashMismatchCount,
			fmt.Sprintf("(+ total links: %v)", r.HashMismatchCount+totalLinks))
		s = statStr(s, "Total hash list searches", r.InoSeqSearchCount)
		avgItersPerSearch := "N/A"
		if r.InoSeqIterationCount > 0 {
			avg := float64(r.InoSeqIterationCount) / float64(r.InoSeqSearchCount)
			avgItersPerSearch = fmt.Sprintf("%.1f", avg)
		}
		s = statStr(s, "Total hash list iterations", r.InoSeqIterationCount,
			fmt.Sprintf("(avg per search: %v)", avgItersPerSearch))
		s = statStr(s, "Total equal comparisons", r.EqualComparisonCount)
		s = statStr(s, "Total digests computed", r.DigestComputedCount)
		if r.FailedLinkChtimesCount > 0 {
			s = statStr(s, "Failed link Chtimes", r.FailedLinkChtimesCount)
		}
		if r.FailedLinkChownCount > 0 {
			s = statStr(s, "Failed link Chown", r.FailedLinkChownCount)
		}
	}

	if r.Opts.DebugLevel > 1 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		s = statStr(s, "Mem Alloc", Humanize(m.Alloc))
		s = statStr(s, "Mem Sys", Humanize(m.Sys))
		s = statStr(s, "Num live objects", m.Mallocs-m.Frees)
	}
	printSlices(s)

	if r.Opts.DebugLevel > 2 {
		fmt.Printf("\nOptions%+v\n", r.Opts)
	}
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

// Humanize returns a string with bytecount "humanized" to a shortened amount
func Humanize(n uint64) string {
	// -1 precision removes trailing zeros
	return HumanizeWithPrecision(n, -1)
}

// HumanizeWithPrecision allows providing FormatFloat precision value
func HumanizeWithPrecision(n uint64, prec int) string {
	var s string
	var m string
	decimals := 1000.0
	if prec > -1 {
		decimals = math.Pow10(prec)
	}
	F := func(N uint64, div float64) string {
		reduced := float64(N) / div
		rounded := math.Round(reduced*decimals) / decimals
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

// humanizeParens returns the humanized number count as a string surrounded by
// parens
func humanizeParens(n uint64) string {
	return fmt.Sprintf("(%v)", Humanize(n))
}

// HumanizedUint64 converts humanized size strings like "1k" into an unsigned
// int64 (ie. "1k" -> 1024)
func HumanizedUint64(s string) (uint64, error) {
	s = strings.ToLower(s)
	mult := map[string]uint64{
		"k": 1 << 10, // 1024
		"m": 1 << 20, // 1024**2
		"g": 1 << 30, // 1024**3
		"t": 1 << 40, // 1024**4
		"p": 1 << 50, // 1024**5
	}
	// If the last character is not one of the multiplier letters, try
	// parsing as a normal number string
	c := s[len(s)-1:] // last char
	if _, ok := mult[c]; !ok {
		n, err := strconv.ParseUint(s, 10, 64)
		return n, err
	}
	// Otherwise, parse the prefix digits and apply the multiplier
	n, err := strconv.ParseUint(s[:len(s)-1], 10, 64)
	if err != nil {
		return n, err
	}
	if n > (math.MaxUint64 / mult[c]) {
		return 0, fmt.Errorf("Size value (%v) is too large for 64 bits", s)
	}
	return n * mult[c], nil
}
