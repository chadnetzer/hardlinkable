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
	"hardlinkable/internal/inode"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/xattr"
)

const testdata0 = ""
const testdata1a = "A"
const testdata1b = "B"
const testdata2a = "aa"
const testdata2b = "bb"

// Algorithm from http://www.quickperm.org/
// Output of emptyset is nil
func permutations(a []string) <-chan []string {
	out := make(chan []string)
	go func() {
		defer close(out)

		if len(a) == 0 {
			return
		} else {
			// Output initial (ie. non-permuted) ordering as first result
			r := make([]string, len(a))
			copy(r, a)
			out <- r
		}

		// Init permutation index array
		N := len(a)
		p := make([]int, N+1)
		for i := 0; i < N+1; i++ {
			p[i] = i
		}
		// Loop over a, swapping and updating perm array
		i := 1
		for i < N {
			var j int
			p[i]--
			j = (i % 2) * p[i]
			a[j], a[i] = a[i], a[j]
			for i = 1; p[i] == 0; i++ {
				p[i] = i
			}
			r := make([]string, len(a))
			copy(r, a)
			out <- r
		}
	}()
	return out
}

func TestPermutations(t *testing.T) {
	testData := []struct {
		p    []string
		want int
	}{
		{[]string{}, 0},
		{[]string{"a"}, 1},
		{[]string{"a", "b"}, 2},
		{[]string{"a", "b", "c"}, 6},
		{[]string{"a", "b", "c", "d"}, 24},
		{[]string{"a", "b", "c", "d", "e"}, 120},
	}

	for _, d := range testData {
		seen := make(map[string]struct{})
		for v := range permutations(d.p) {
			// convert v to string (for storing to map for testing)
			j := strings.Join(v, " ")
			if _, ok := seen[j]; ok {
				t.Errorf("Repeated permutation found: %v\n", v)
			}
			seen[j] = struct{}{}
		}
		if len(seen) != d.want {
			t.Errorf("Expected %v permutations for %v, got: %v", d.want, d.p, len(seen))
		}
	}
}

// Does not output the empty set
func powerset(s []string) <-chan []string {
	out := make(chan []string)
	go func() {
		defer close(out)
		prevSets := [][]string{[]string{}}
		for _, s := range s {
			for _, p := range prevSets {
				newSet := append([]string{}, p...)
				newSet = append(newSet, s)
				prevSets = append(prevSets, newSet)
				out <- newSet
			}
		}
	}()
	return out
}

func TestPowersets(t *testing.T) {
	testData := []struct {
		p    []string
		want int
	}{
		{[]string{}, 0},
		{[]string{"a"}, 1},
		{[]string{"a", "b"}, 3},
		{[]string{"a", "b", "c"}, 7},
		{[]string{"a", "b", "c", "d"}, 15},
		{[]string{"a", "b", "c", "d", "e"}, 31},
	}

	for _, d := range testData {
		N := len(d.p)
		lengthCounts := map[int]int{}
		seen := map[string]struct{}{}
		for v := range powerset(d.p) {
			// convert v to string (for storing to map for testing)
			j := strings.Join(v, " ")
			if _, ok := seen[j]; ok {
				t.Errorf("Repeated powerset found: %v\n", v)
			}
			seen[j] = struct{}{}
			lengthCounts[len(v)] += 1
		}
		if len(seen) != d.want {
			t.Errorf("Expected %v powersets for %v, got: %v", d.want, d.p, len(seen))
		}

		var z big.Int
		possibleLengthCounts := map[int]int{}
		for i := 1; i <= N; i++ {
			// Compute binomial coeffs (ie. Pascal's triangle) for input length.
			n := int(z.Binomial(int64(N), int64(i)).Int64())
			possibleLengthCounts[i] = n
		}
		if !reflect.DeepEqual(lengthCounts, possibleLengthCounts) {
			t.Errorf("Incorrect powerset lengths: expected %v, got %v", possibleLengthCounts, lengthCounts)
		}
	}
}

func powersetPerms(s []string) <-chan []string {
	out := make(chan []string)
	go func() {
		defer close(out)
		for v := range powerset(s) {
			for p := range permutations(v) {
				out <- p
			}
		}
	}()
	return out
}

func TestPowersetPerms(t *testing.T) {
	testData := []struct {
		p    []string
		want int
	}{
		// "want" lengths from https://oeis.org/A007526
		// a(n) = n*(a(n) + 1)
		{[]string{"a"}, 1},
		{[]string{"a", "b"}, 4},
		{[]string{"a", "b", "c"}, 15},
		{[]string{"a", "b", "c", "d"}, 64},
		{[]string{"a", "b", "c", "d", "e"}, 325},
	}

	for _, d := range testData {
		seen := map[string]struct{}{}
		for v := range powersetPerms(d.p) {
			j := strings.Join(v, "")
			if _, ok := seen[j]; ok {
				t.Errorf("Repeated powersetPerm found: %v\n", v)
			}
			seen[j] = struct{}{}
		}
		if len(seen) != d.want {
			t.Errorf("Expected %v powersetPerms for %v, got: %v", d.want, d.p, len(seen))
		}
	}
}

func setUp(name string, t *testing.T) string {
	topdir, err := ioutil.TempDir("", "hardlinkable")
	if err != nil {
		t.Fatalf("Couldn't create temp dir for %v tests: %v", topdir, err)
	}

	if os.Chdir(topdir) != nil {
		t.Fatalf("Couldn't chdir to temp dir for %v tests", topdir)
	}

	return topdir
}

type stringSet map[string]struct{}

func newSet(args ...string) stringSet {
	set := make(stringSet)
	for _, s := range args {
		set[s] = struct{}{}
	}
	return set
}

// Keeps it simple and doesn't worry about performance
func intersection(a, b stringSet) stringSet {
	r := make(stringSet)
	for s := range a {
		if _, ok := b[s]; ok {
			r[s] = struct{}{}
		}
	}
	return r
}

func (s stringSet) asSlice() []string {
	r := make([]string, 0)
	for k := range s {
		r = append(r, k)
	}
	return r
}

func simpleRun(name string, t *testing.T, opts Options, numLinkPaths int, dirs ...string) *Results {
	result, err := Run(dirs, []string{}, opts)
	if err != nil {
		t.Errorf("%v: Run() returned error: %v\n", name, err)
	}
	if !result.RunSuccessful {
		t.Errorf("%v: Run() was not successful (aborted early)", name)
	}
	if len(result.LinkPaths) != numLinkPaths {
		t.Errorf("%v: len(LinkPaths) expected %v:  got: %v\n", name, numLinkPaths, len(result.LinkPaths))
	}
	return &result
}

type pathContents map[string]string
type existingLinks map[string][]string

// provided with a map of filenames:content, create the files
func simpleFileMaker(t *testing.T, m pathContents) {
	now := time.Now()
	for name, content := range m {
		dirname := path.Dir(name)
		os.MkdirAll(dirname, 0755)
		if err := ioutil.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatalf("Couldn't create test file '%v'", name)
		}
		if err := os.Chtimes(name, now, now); err != nil {
			t.Fatalf("Couldn't Chtimes() on test file '%v'", name)
		}
	}
}

// provided with a map of filenames:content, create the files
func simpleLinkMaker(t *testing.T, src string, dsts ...string) {
	for _, dst := range dsts {
		dirname := path.Dir(dst)
		os.MkdirAll(dirname, 0755)
		if err := os.Link(src, dst); err != nil {
			t.Fatalf("Couldn't create test link '%v' to '%v'", src, dst)
		}
	}
}

func nlinkVal(pathname string) uint32 {
	l, err := os.Lstat(pathname)
	if err != nil {
		return 0
	}
	stat_t, ok := l.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint32(stat_t.Nlink)
}

func inoVal(pathname string) uint64 {
	l, err := os.Lstat(pathname)
	if err != nil {
		return 0
	}
	stat_t, ok := l.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint64(stat_t.Ino)
}

func verifyLinkPaths(name string, t *testing.T, r *Results, paths []string) {
	if len(paths) == 0 && len(r.LinkPaths) > 0 {
		t.Errorf("%v: Expected empty LinkedPaths, got: %v\n", name, r.LinkPaths)
		return
	}
	if len(paths) == 0 {
		return
	}
	pathsSet := newSet(paths...)
	for _, l := range r.LinkPaths {
		lSet := newSet(l...)
		overlap := intersection(pathsSet, lSet)
		if len(overlap) == len(paths) {
			return
		}
	}
	t.Errorf("%v: Couldn't find expected LinkedPaths in results: %v\n", name, paths)
}

func verifyInodeCounts(name string, t *testing.T, r *Results, inoRemovedCount int64, inoRemovedBytes uint64, nlinkCount uint32, filenames ...string) {
	if r.InodeRemovedCount != inoRemovedCount {
		t.Errorf("%v: InodeRemovedCount expected: %v, got: %v\n", name, inoRemovedCount, r.InodeRemovedCount)
	}
	if r.InodeRemovedByteAmount != inoRemovedBytes {
		t.Errorf("%v: InodeRemovedByteAmount expected: %v, got: %v\n", name, inoRemovedBytes, r.InodeRemovedByteAmount)
	}
	for _, filename := range filenames {
		if nlinkVal(filename) != nlinkCount {
			t.Errorf("%v: Inode nlink count for '%v' expected: %v, got: %v\n", name, filename, nlinkCount, nlinkVal(filename))
		}
	}
}

func verifyContents(name string, t *testing.T, m pathContents) {
	for pathname, content := range m {
		readContent, err := ioutil.ReadFile(pathname)
		if err != nil {
			t.Fatalf("%v: Couldn't read test file contents: %v\n", name, pathname)
		}
		if content != string(readContent) {
			t.Errorf("%v: Mismatched content for: %v, expected: %v, got: %v\n",
				name, pathname, content, string(readContent))
		}
	}
}

func numNonEmpty(l [][]string) int {
	var count int
	for _, s := range l {
		if len(s) > 0 {
			count++
		}
	}
	return count
}

func TestRunLinkingTableTests(t *testing.T) {
	tsts := []struct {
		name            string
		opts            Options
		c               pathContents
		l               existingLinks
		linkedPaths     [][]string
		inoRemovedCount int
		inoRemovedBytes int
		nlinkCounts     map[int][]string
	}{
		{
			name:            "testname: 'Linking Disabled'",
			opts:            SetupOptions(),
			c:               pathContents{"f1": "X", "f2": "X"},
			l:               existingLinks{"f2": []string{"f3"}},
			linkedPaths:     [][]string{[]string{"f1", "f2"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts: map[int][]string{
				1: []string{"f1"},
				2: []string{"f2", "f3"},
			},
		},
		{
			name:            "testname: 'No Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{},
			l:               existingLinks{},
			linkedPaths:     [][]string{},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int][]string{},
		},
		{
			name:            "testname: 'One File'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X"},
			l:               existingLinks{},
			linkedPaths:     [][]string{},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int][]string{1: []string{"f1"}},
		},
		{
			name:            "testname: 'Two Equal Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "X"},
			l:               existingLinks{},
			linkedPaths:     [][]string{[]string{"f1", "f2"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int][]string{2: []string{"f1", "f2"}},
		},
		{
			name:            "testname: 'Two Unequal Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "Y"},
			l:               existingLinks{},
			linkedPaths:     [][]string{[]string{}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int][]string{1: []string{"f1", "f2"}},
		},
		{
			name:            "testname: 'Two Equal Files One Existing Link'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "X"},
			l:               existingLinks{"f2": []string{"f3"}},
			linkedPaths:     [][]string{[]string{"f1", "f2"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int][]string{3: []string{"f1", "f2", "f3"}},
		},
		{
			name:            "testname: 'Two Groups of Equal Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "X", "f3": "YY", "f4": "YY", "f5": "YY"},
			l:               existingLinks{},
			linkedPaths:     [][]string{[]string{"f1", "f2"}, []string{"f3", "f4", "f5"}},
			inoRemovedCount: 3,
			inoRemovedBytes: 5,
			nlinkCounts: map[int][]string{
				2: []string{"f1", "f2"},
				3: []string{"f3", "f4", "f5"},
			},
		},
		{
			name:            "testname: 'One File With Two Existing Links'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X"},
			l:               existingLinks{"f1": []string{"f2", "f3"}},
			linkedPaths:     [][]string{[]string{}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int][]string{3: []string{"f1", "f2", "f3"}},
		},
		{
			name:            "testname: 'Two Files With One Existing Link'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "X"},
			l:               existingLinks{"f1": []string{"f3"}},
			linkedPaths:     [][]string{[]string{"f1", "f2"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int][]string{3: []string{"f1", "f2", "f3"}},
		},
		{
			name:            "testname: 'Equal Filenames Only'",
			opts:            SetupOptions(LinkingEnabled, SameName),
			c:               pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l:               existingLinks{"A/f1": []string{"A/f0", "A/f3", "B/a0", "B/f100"}},
			linkedPaths:     [][]string{[]string{"A/f1", "B/f1"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int][]string{6: []string{"A/f1", "B/f1", "A/f0", "A/f3", "B/a0", "B/f100"}},
		},
		{
			name: "testname: 'Equal Filenames No Removed Inodes test'",
			//opts: SetupOptions(LinkingEnabled, SameName),
			opts: SetupOptions(LinkingEnabled, SameName),
			c:    pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l: existingLinks{
				"A/f1": []string{"A/f0", "A/f3"},
			},
			linkedPaths:     [][]string{[]string{"A/f1", "B/f1"}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts: map[int][]string{
				4: []string{"A/f1", "B/f1", "A/f0", "A/f3"},
			},
		},
		{
			name: "testname: 'Equal Filenames No Removed Inodes'",
			opts: SetupOptions(LinkingEnabled, SameName),
			c:    pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l: existingLinks{
				"A/f1": []string{"A/f0", "A/f3"},
				"B/f1": []string{"B/a0", "B/f100"},
			},
			linkedPaths:     [][]string{[]string{"A/f1", "B/f1"}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts: map[int][]string{
				4: []string{"A/f1", "B/f1"},
			},
		},
	}
	for _, tst := range tsts {
		func() {
			topdir := setUp("Run", t)
			defer os.RemoveAll(topdir)

			simpleFileMaker(t, tst.c)
			for src, dsts := range tst.l {
				simpleLinkMaker(t, src, dsts...)
			}
			result := simpleRun(tst.name, t, tst.opts, numNonEmpty(tst.linkedPaths), ".")
			for _, l := range tst.linkedPaths {
				verifyLinkPaths(tst.name, t, result, l)
			}
			// Note the values of the result, reporting what could be done, and the
			// difference from the nlinks read off the disk (which should be
			// unaltered, since linking was disabled)
			for k, v := range tst.nlinkCounts {
				verifyInodeCounts(tst.name, t, result,
					int64(tst.inoRemovedCount),
					uint64(tst.inoRemovedBytes),
					uint32(k), v...)
			}
			verifyContents(tst.name, t, tst.c)
		}()
	}
}

func TestRunLinkedFileOutsideOfWalk(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'LinkedFileOutsideOfWalk'"

	// Test linked file outside of walked tree
	m := pathContents{"A/f1": "X"}
	simpleFileMaker(t, m)
	simpleLinkMaker(t, "A/f1", "B/f2")
	result := simpleRun(name, t, opts, 0, "A")
	verifyInodeCounts(name, t, result, 0, 0, 2, "A/f1")
	m["B/f2"] = "X"
	verifyContents(name, t, m)
	if result.PrevLinkCount != 0 {
		t.Errorf("Out of tree links counted, expected 0, got %v\n", result.PrevLinkCount)
	}
	if result.PrevLinkedByteAmount != 0 {
		t.Errorf("Out of tree linked bytes, expected 0, got %v\n", result.PrevLinkedByteAmount)
	}
}

func TestRunTwoDifferentTimes(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'Two Different File Times'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	now := time.Now()
	then := now.AddDate(-1, 0, 0)
	if err := os.Chtimes("f2", then, then); err != nil {
		t.Fatalf("Failure to set time on test file: 'f2'\n")
	}
	result := simpleRun(name, t, opts, 0, ".")
	verifyLinkPaths(name, t, result, []string{})
	verifyInodeCounts(name, t, result, 0, 0, 1, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunTwoDifferentTimesIgnoreTime(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, IgnoreTime)

	name := "testname: 'Two Unequal File Times w/ IgnoreTime'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	now := time.Now()
	then := now.AddDate(-1, 0, 0)
	if err := os.Chtimes("f2", then, then); err != nil {
		t.Fatalf("Failure to set time on test file: 'f2'\n")
	}
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunIgnorePerms(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, IgnorePerms)

	name := "testname: 'Two Unequal File Modes w/ IgnorePerms'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := os.Chmod("f1", 0644); err != nil {
		t.Fatalf("Couldn't set file 'f1' mode to '0644': %v", err)
	}
	if err := os.Chmod("f2", 0755); err != nil {
		t.Fatalf("Couldn't set file 'f2' mode to '0755': %v", err)
	}
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunExcludeFiles(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)
	opts.FileExcludes = append(opts.FileExcludes, `.*\.ext$`, `^prefix_.*`)

	name := "testname: 'Exclude Files'"

	m := pathContents{"f1": "X", "f2": "X", "f3.ext": "X", "prefix_f4": "X"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunExcludeDirs(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)
	opts.DirExcludes = append(opts.DirExcludes, `^A.*`, `.*B$`)

	name := "testname: 'Exclude Dirs'"

	m := pathContents{"Aetc/f1": "X", "preB/f2": "X", "etcA/f1": "X", "Bpre/f2": "X"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"etcA/f1", "Bpre/f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "etcA/f1", "Bpre/f2")
	verifyContents(name, t, m)
}

func TestRunIncludeFiles(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)
	opts.FileIncludes = append(opts.FileIncludes, `.*\.ext$`, `^prefix_.*`)

	name := "testname: 'Include Files'"

	m := pathContents{"f1": "X", "f2": "X", "f3.ext": "X", "prefix_f4": "X"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f3.ext", "prefix_f4"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f3.ext", "prefix_f4")
	verifyContents(name, t, m)
}

func TestRunReincludeExcludedFiles(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)
	opts.FileExcludes = append(opts.FileExcludes, `.*\.ext$`, `^prefix_.*`)
	opts.FileIncludes = append(opts.FileIncludes, `^prefix_.*`)

	name := "testname: 'Include Files'"

	m := pathContents{"f1": "X", "f2": "X", "f3.ext": "X", "prefix_f4": "X"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2", "prefix_f4"})
	verifyInodeCounts(name, t, result, 2, 2, 3, "f1", "f2", "prefix_f4")
	verifyContents(name, t, m)
}

func TestRunMinMaxSize(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, MinFileSize(2), MaxFileSize(2))

	name := "testname: 'Min/Max File Sizes'"

	m := pathContents{"f1": "X", "f2": "X", "f3": "YY", "f4": "YY", "f5": "ZZZ", "f6": "ZZZ"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f3", "f4"})
	verifyInodeCounts(name, t, result, 1, 2, 2, "f3", "f4")
	verifyContents(name, t, m)
}

func TestRunExcludedMinMaxSize(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, MinFileSize(2), MaxFileSize(2))

	name := "testname: 'Excluded Min/Max File Sizes'"

	m := pathContents{"f1": "X", "f2": "X", "f5": "ZZZ", "f6": "ZZZ"}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 0, ".")
	verifyLinkPaths(name, t, result, []string{})
	verifyInodeCounts(name, t, result, 0, 0, 1, "f1", "f2", "f5", "f6")
	verifyContents(name, t, m)
}

func TestRunZeroMinSize(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, MinFileSize(0), MaxFileSize(1))

	name := "testname: 'Zero Min File Size'"

	m := pathContents{"f1": "", "f2": ""}
	simpleFileMaker(t, m)
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 0, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunCrossedMinMaxSize(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	const min = 2
	const max = 1
	opts := SetupOptions(LinkingEnabled, MinFileSize(min), MaxFileSize(max))

	result, err := Run([]string{"."}, []string{}, opts)
	if err == nil {
		t.Errorf("Run succeeded with incorrect min(%v) and max(%v) size options\n", min, max)
	}
	if result.RunSuccessful {
		t.Errorf("Run result was 'successful' with improper min(%v) and max(%v) size options\n", min, max)
	}
}

func TestRunEqualXattrs(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'Equal Xattrs'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'foo':'bar'\n")
	}
	if err := xattr.Set("f2", "foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'foo':'bar'\n")
	}
	if err := xattr.Set("f1", "baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'baz':'abc'\n")
	}
	if err := xattr.Set("f2", "baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'baz':'abc'\n")
	}

	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunUnequalXattrs(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'Unequal Xattrs'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'foo':'bar'\n")
	}
	if err := xattr.Set("f2", "baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'baz':'abc'\n")
	}

	result := simpleRun(name, t, opts, 0, ".")
	verifyLinkPaths(name, t, result, []string{})
	verifyInodeCounts(name, t, result, 0, 0, 1, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunEqualXattrsIgnoreXattr(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, IgnoreXattr)

	name := "testname: 'Unequal Xattrs w/ IgnoreXattr'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'foo':'bar'\n")
	}
	if err := xattr.Set("f2", "foo", []byte{'x', 'y', 'z'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'foo':'xyz'\n")
	}
	if err := xattr.Set("f2", "baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'baz':'abc'\n")
	}

	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, []string{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunLinearVsDigestSearch(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	// Note - linking disabled to allow re-running multiple times
	opts := SetupOptions(LinkingDisabled)

	m := pathContents{
		"f1": "X", "f2": "X",
		"f3": "YY", "f4": "YY",
		"f5": "ZZZ", "f6": "ZZZ",
		"a1": "A", "a2": "A", "a3": "A", "a4": "A", "a5": "A",
		"a6": "A", "a7": "A", "a8": "A", "a9": "A", "a10": "A",
		"b1": "B", "b2": "B", "b3": "B", "b4": "B", "b5": "B",
		"b6": "B", "b7": "B", "b8": "B", "b9": "B", "b10": "B",
		"c1": "C", "c2": "C", "c3": "C", "c4": "C", "c5": "C",
		"c6": "C", "c7": "C", "c8": "C", "c9": "C", "c10": "C",
	}
	simpleFileMaker(t, m)

	// Confirm that results match for different max linear search lengths
	for i := -1; i < 12; i++ {
		name := fmt.Sprintf("testname: 'Linear Vs Digest Search' val=%v", i)
		opts.SearchThresh = i
		result := simpleRun(name, t, opts, 6, ".")
		verifyLinkPaths(name, t, result, []string{"f1", "f2"})
		verifyLinkPaths(name, t, result, []string{"f3", "f4"})
		verifyLinkPaths(name, t, result, []string{"f5", "f6"})
		verifyInodeCounts(name, t, result, 30, 33, 1)
		verifyContents(name, t, m)
	}
}

func TestMaxNlinks(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	m := pathContents{"f1": "X"}
	simpleFileMaker(t, m)

	N := inode.MaxNlinkVal("f1")
	if N > (1<<15 - 1) {
		t.Skip("Skipping MaxNlink test because Nlink max is greater than 32767")
	}
	if testing.Short() {
		t.Skip("Skipping MaxNlink test in short mode")
	} else {
		t.Log("Use -short option to skip MaxNlinks test")
	}

	opts := SetupOptions(LinkingEnabled)

	m = pathContents{}
	for i := 0; i < int(N+100); i++ {
		filename := fmt.Sprintf("n%v", i)
		m[filename] = "Y"
	}
	simpleFileMaker(t, m)

	name := "testname: 'MaxNlinks'"
	simpleRun(name, t, opts, 2, ".")
	verifyContents(name, t, m)

	counts := make(map[int]int)
	for i := 0; i < int(2*N+100); i++ {
		filename := fmt.Sprintf("n%v", i)
		n := int(inoVal(filename))
		counts[n] += 1
	}

	foundNlinks := []int{}
	for _, v := range counts {
		foundNlinks = append(foundNlinks, v)
	}
	sort.Ints(foundNlinks)
	expectedNlinks := []int{100, int(N), int(N)}

	for i, v := range foundNlinks {
		if v != expectedNlinks[i] {
			t.Errorf("Nlink and leftover counts, expected: %v, got: %v", expectedNlinks, foundNlinks)
			break
		}
	}
}