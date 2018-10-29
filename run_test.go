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
	"io/ioutil"
	"math/big"
	"math/rand"
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

// ShuffleString returns a random shuffle of a given string (for test case
// generation uses; complex unicode will likely confuse it)
func ShuffleString(s string) string {
	b := []rune(s)

	dest := make([]rune, len(b))
	perm := rand.Perm(len(b))
	for i, v := range perm {
		dest[v] = b[i]
	}
	return string(dest)
}

// Algorithm from http://www.quickperm.org/
// Output of emptyset is nil
func permutations(a []string) <-chan []string {
	out := make(chan []string)
	go func() {
		defer close(out)

		if len(a) == 0 {
			return
		}
		// Output initial (ie. non-permuted) ordering as first result
		r := make([]string, len(a))
		copy(r, a)
		out <- r

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
		{[]string{"c", "b", "a"}, 6},
		{[]string{"b", "a", "c"}, 6},
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
		{[]string{"b", "c", "a"}, 7},
		{[]string{"c", "a", "b"}, 7},
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
			lengthCounts[len(v)]++
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
		{[]string{"b", "a"}, 4},
		{[]string{"a", "b", "c"}, 15},
		{[]string{"c", "a", "b"}, 15},
		{[]string{"b", "c", "a"}, 15},
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
	result, err := Run(dirs, opts)
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

type paths []string
type pathContents map[string]string // pathname:contents
type existingLinks map[string]paths // pathname:pathnames
type linkedPaths []paths
type linkedPathsOptions []linkedPaths

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
	statT, ok := l.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint32(statT.Nlink)
}

func verifyLinkPaths(name string, t *testing.T, r *Results, p paths) bool {
	if len(p) == 0 && len(r.LinkPaths) > 0 {
		t.Errorf("%v: Expected empty LinkPaths, got: %v\n", name, r.LinkPaths)
		return false
	}
	if len(p) == 0 {
		return true
	}
	pathsSet := newSet(p...)
	for _, l := range r.LinkPaths {
		lSet := newSet(l...)
		overlap := intersection(pathsSet, lSet)
		if len(overlap) == len(p) {
			return true
		}
	}
	return false
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

func numNonEmpty(lpo linkedPathsOptions) int {
	var count int
	if len(lpo) == 0 {
		return 0
	}
	for _, s := range lpo[0] {
		if len(s) > 0 {
			count++
		}
	}
	return count
}

func TestRunLinkingTable(t *testing.T) {
	tsts := []struct {
		name            string
		opts            Options
		c               pathContents
		l               existingLinks
		lpo             linkedPathsOptions
		inoRemovedCount int
		inoRemovedBytes int
		nlinkCounts     map[int]paths
	}{
		{
			name: "testname: 'Linking Disabled'",
			opts: SetupOptions(),
			c:    pathContents{"f1": "X", "f2": "X"},
			l:    existingLinks{"f2": paths{"f3"}},
			lpo: linkedPathsOptions{
				linkedPaths{paths{"f1", "f2"}},
				linkedPaths{paths{"f1", "f3"}},
			},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts: map[int]paths{
				1: paths{"f1"},
				2: paths{"f2", "f3"},
			},
		},
		{
			name:            "testname: 'No Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{},
			l:               existingLinks{},
			lpo:             linkedPathsOptions{linkedPaths{paths{}}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int]paths{},
		},
		{
			name:            "testname: 'One File'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X"},
			l:               existingLinks{},
			lpo:             linkedPathsOptions{linkedPaths{paths{}}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int]paths{1: paths{"f1"}},
		},
		{
			name:            "testname: 'Two Equal Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "X"},
			l:               existingLinks{},
			lpo:             linkedPathsOptions{linkedPaths{paths{"f1", "f2"}}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int]paths{2: paths{"f1", "f2"}},
		},
		{
			name:            "testname: 'Two Unequal Files'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X", "f2": "Y"},
			l:               existingLinks{},
			lpo:             linkedPathsOptions{linkedPaths{paths{}}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int]paths{1: paths{"f1", "f2"}},
		},
		{
			name: "testname: 'Two Equal Files One Existing Link'",
			opts: SetupOptions(LinkingEnabled),
			c:    pathContents{"f1": "X", "f2": "X"},
			l:    existingLinks{"f2": paths{"f3"}},
			lpo: linkedPathsOptions{
				linkedPaths{paths{"f1", "f2"}},
				linkedPaths{paths{"f1", "f3"}},
			},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int]paths{3: paths{"f1", "f2", "f3"}},
		},
		{
			name: "testname: 'Two Groups of Equal Files'",
			opts: SetupOptions(LinkingEnabled),
			c:    pathContents{"f1": "X", "f2": "X", "f3": "YY", "f4": "YY", "f5": "YY"},
			l:    existingLinks{},
			lpo: linkedPathsOptions{
				linkedPaths{paths{"f1", "f2"}, paths{"f3", "f4", "f5"}},
			},
			inoRemovedCount: 3,
			inoRemovedBytes: 5,
			nlinkCounts: map[int]paths{
				2: paths{"f1", "f2"},
				3: paths{"f3", "f4", "f5"},
			},
		},
		{
			name:            "testname: 'One File With Two Existing Links'",
			opts:            SetupOptions(LinkingEnabled),
			c:               pathContents{"f1": "X"},
			l:               existingLinks{"f1": paths{"f2", "f3"}},
			lpo:             linkedPathsOptions{linkedPaths{paths{}}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts:     map[int]paths{3: paths{"f1", "f2", "f3"}},
		},
		{
			name: "testname: 'Two Files With One Existing Link'",
			opts: SetupOptions(LinkingEnabled),
			c:    pathContents{"f1": "X", "f2": "X"},
			l:    existingLinks{"f1": paths{"f3"}},
			lpo: linkedPathsOptions{
				linkedPaths{paths{"f1", "f2"}},
				linkedPaths{paths{"f1", "f3"}},
			},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int]paths{3: paths{"f1", "f2", "f3"}},
		},
		{
			name:            "testname: 'Equal Filenames Only'",
			opts:            SetupOptions(LinkingEnabled, SameName),
			c:               pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l:               existingLinks{"A/f1": paths{"A/f0", "A/f3", "B/a0", "B/f100"}},
			lpo:             linkedPathsOptions{linkedPaths{paths{"A/f1", "B/f1"}}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts:     map[int]paths{6: paths{"A/f1", "B/f1", "A/f0", "A/f3", "B/a0", "B/f100"}},
		},
		{
			name: "testname: 'Equal Filenames No Removed Inodes test'",
			opts: SetupOptions(LinkingEnabled, SameName),
			c:    pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l: existingLinks{
				"A/f1": paths{"A/f0", "A/f3"},
			},
			lpo:             linkedPathsOptions{linkedPaths{paths{"A/f1", "B/f1"}}},
			inoRemovedCount: 1,
			inoRemovedBytes: 1,
			nlinkCounts: map[int]paths{
				4: paths{"A/f1", "B/f1", "A/f0", "A/f3"},
			},
		},
		{
			name: "testname: 'Equal Filenames No Removed Inodes'",
			opts: SetupOptions(LinkingEnabled, SameName),
			c:    pathContents{"A/f1": "X", "B/f1": "X", "B/f2": "X"},
			l: existingLinks{
				"A/f1": paths{"A/f0", "A/f3"},
				"B/f1": paths{"B/a0", "B/f100"},
			},
			lpo:             linkedPathsOptions{linkedPaths{paths{"A/f1", "B/f1"}}},
			inoRemovedCount: 0,
			inoRemovedBytes: 0,
			nlinkCounts: map[int]paths{
				4: paths{"A/f1", "B/f1"},
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
			result := simpleRun(tst.name, t, tst.opts, numNonEmpty(tst.lpo), ".")
			verified := false
		VerifiedTest:
			for _, lp := range tst.lpo {
				for _, l := range lp {
					if verifyLinkPaths(tst.name, t, result, l) {
						verified = true
						break VerifiedTest
					}
				}
			}
			if !verified {
				t.Errorf("%v: Couldn't find expected LinkPaths: %v in results: %v\n", tst.name, tst.lpo, result.LinkPaths)
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
	if result.ExistingLinkCount != 0 {
		t.Errorf("Out of tree links counted, expected 0, got %v\n", result.ExistingLinkCount)
	}
	if result.ExistingLinkByteAmount != 0 {
		t.Errorf("Out of tree linked bytes, expected 0, got %v\n", result.ExistingLinkByteAmount)
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
	verifyLinkPaths(name, t, result, paths{})
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
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunIgnorePerm(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, IgnorePerm)

	name := "testname: 'Two Unequal File Modes w/ IgnorePerm'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := os.Chmod("f1", 0644); err != nil {
		t.Fatalf("Couldn't set file 'f1' mode to '0644': %v", err)
	}
	if err := os.Chmod("f2", 0755); err != nil {
		t.Fatalf("Couldn't set file 'f2' mode to '0755': %v", err)
	}
	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
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
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
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
	verifyLinkPaths(name, t, result, paths{"etcA/f1", "Bpre/f2"})
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
	verifyLinkPaths(name, t, result, paths{"f3.ext", "prefix_f4"})
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
	verifyLinkPaths(name, t, result, paths{"f1", "f2", "prefix_f4"})
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
	verifyLinkPaths(name, t, result, paths{"f3", "f4"})
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
	verifyLinkPaths(name, t, result, paths{})
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
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 0, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunCrossedMinMaxSize(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	const min = 2
	const max = 1
	opts := SetupOptions(LinkingEnabled, MinFileSize(min), MaxFileSize(max))

	result, err := Run([]string{"."}, opts)
	if err == nil {
		t.Errorf("Run succeeded with incorrect min(%v) and max(%v) size options\n", min, max)
	}
	if result.RunSuccessful {
		t.Errorf("Run result was 'successful' with improper min(%v) and max(%v) size options\n", min, max)
	}
}

func TestRunEqualXAttrs(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'Equal Xattrs'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "user.foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'user.foo':'bar'  %v\n", err)
	}
	if err := xattr.Set("f2", "user.foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'user.foo':'bar'  %v\n", err)
	}
	if err := xattr.Set("f1", "user.baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'user.baz':'abc'  %v\n", err)
	}
	if err := xattr.Set("f2", "user.baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'user.baz':'abc'  %v\n", err)
	}

	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
	verifyInodeCounts(name, t, result, 1, 1, 2, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunUnequalXAttrs(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled)

	name := "testname: 'Unequal Xattrs'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "user.foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'user.foo':'bar'  %v\n", err)
	}
	if err := xattr.Set("f2", "user.baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'user.baz':'abc'  %v\n", err)
	}

	result := simpleRun(name, t, opts, 0, ".")
	verifyLinkPaths(name, t, result, paths{})
	verifyInodeCounts(name, t, result, 0, 0, 1, "f1", "f2")
	verifyContents(name, t, m)
}

func TestRunEqualXAttrsIgnoreXAttr(t *testing.T) {
	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, IgnoreXAttr)

	name := "testname: 'Unequal Xattrs w/ IgnoreXattr'"

	m := pathContents{"f1": "X", "f2": "X"}
	simpleFileMaker(t, m)
	if err := xattr.Set("f1", "user.foo", []byte{'b', 'a', 'r'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f1', 'user.foo':'bar'  %v\n", err)
	}
	if err := xattr.Set("f2", "user.foo", []byte{'x', 'y', 'z'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'user.foo':'xyz'  %v\n", err)
	}
	if err := xattr.Set("f2", "user.baz", []byte{'a', 'b', 'c'}); err != nil {
		t.Fatalf("Couldn't set xattr on test file: 'f2', 'user.baz':'abc'  %v\n", err)
	}

	result := simpleRun(name, t, opts, 1, ".")
	verifyLinkPaths(name, t, result, paths{"f1", "f2"})
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
		verifyLinkPaths(name, t, result, paths{"f1", "f2"})
		verifyLinkPaths(name, t, result, paths{"f3", "f4"})
		verifyLinkPaths(name, t, result, paths{"f5", "f6"})
		verifyInodeCounts(name, t, result, 30, 33, 1)
		verifyContents(name, t, m)
	}
}

type PathnameSet map[string]struct{} // string = pathname
type Clusters []PathnameSet

func newPathnameSet(s string) PathnameSet {
	ps := PathnameSet{}
	ps[s] = struct{}{}
	return ps
}

// Add newPath to the cluster containing prevPath
func (c Clusters) addToCluster(prevPath, newPath string) {
	for _, m := range c {
		if _, ok := m[prevPath]; ok {
			m[newPath] = struct{}{}
			break
		}
	}
}

type randTestVals struct {
	minSize            int
	maxSize            int
	numDirs            int64
	numFiles           int64
	numNewLinks        int64
	numExistingLinks   int64
	numInodes          int64
	numNlinks          int64
	linkPathsBytes     uint64
	existingLinksBytes uint64
	pc                 pathContents        // pathname:contents map
	contentPaths       map[string][]string // contents:[]pathname map
	contentClusters    map[string]Clusters // contents:Clusters

	// A set of all the file contents we've used, and their usage count
	contents map[string]int // contents:fileCount
}

func newRandTestVals() *randTestVals {
	return &randTestVals{
		pc:              make(pathContents),
		contentPaths:    make(map[string][]string),
		contentClusters: make(map[string]Clusters),
		contents:        make(map[string]int),
	}
}

func setupRandTestFiles(t *testing.T, topdir string, samename bool) *randTestVals {
	r := newRandTestVals()

	// Use "go test -count=1" to disable test result caching, otherwise the
	// tests will not run with a new seed.
	seed := time.Now().UnixNano()
	rand.Seed(seed)

	const maxContentLen = (1 << 18)
	const maxContentIndex = maxContentLen - 1

	// Generate a bunch of random bytes
	contentSrc := make([]byte, maxContentLen)
	rand.Read(contentSrc)

	// Setup min/max file sizes
	if samename {
		// Using samename option with excluded files due to size restrictions
		// affects the end result of the linking (because the on-disk files are
		// sorted by nlink, which are not reflected in the cluster sorting.
		// For now, work around by disabling file size restrictions for
		// samename mode.
		r.maxSize = 0
		r.minSize = 0
	} else {
		r.maxSize = rand.Intn(maxContentIndex) + 1
		r.minSize = rand.Intn(r.maxSize)
	}

	dirnameChars := ShuffleString("ABC")
	filenameChars := ShuffleString("abcde")
	for dirs := range powersetPerms(strings.Split(dirnameChars, "")) {
		newDir := true
		for files := range powersetPerms(strings.Split(filenameChars, "")) {
			dirname := strings.Join(dirs, "")
			filename := strings.Join(files, "")
			pathname := path.Join(dirname, filename)

			if err := os.Mkdir(dirname, 0755); err != nil && !os.IsExist(err) {
				t.Fatalf("Couldn't create dirname '%v'", dirname)
			}

			var b []byte
			var s string
			// Each new file can either be new content or repeated
			// content, or a link to an existing path.
			rnd := rand.Float32()
			if len(r.pc) > 0 && rnd > 0.9 {
				// Link to arbitrary exising pathname (can create clusters)
				var oldPathname string
				n := rand.Intn(len(r.pc))
				for k := range r.pc {
					if n == 0 {
						oldPathname = k
						break
					}
					n--
				}

				if err := os.Link(oldPathname, pathname); err != nil {
					t.Fatalf("Couldn't link %v to %v: %v", pathname, oldPathname, err)
				}

				s = r.pc[oldPathname]
				if len(s) >= r.minSize && (r.maxSize == 0 || len(s) <= r.maxSize) {
					r.contentClusters[s].addToCluster(oldPathname, pathname)
				}
			} else {
				rnd := rand.Float32()
				if len(r.contents) > 0 && rnd < 0.25 {
					// Choose arbitrary existing contents
					n := rand.Intn(len(r.contents))
					for k := range r.contents {
						if n == 0 {
							b = []byte(k)
							break
						}
						n--
					}
				} else {
					// Come up with a previously unseen content string
					for {
						// Weight max length towards zero, basically
						// making it more likely to have smaller files
						// than large (but definitely allow large)
						const avgSize = 8192
						var n int
						for {
							n = int(rand.ExpFloat64() * avgSize)
							if n < len(contentSrc) {
								n++
								break
							}
						}
						m := rand.Intn(n)
						b = contentSrc[m:n]

						if _, ok := r.contents[string(b)]; !ok {
							break
						}
					}
				}

				if err := ioutil.WriteFile(pathname, b, 0644); err != nil {
					t.Fatalf("Couldn't write pathname '%v' w/ rnd byte contents", pathname)
				}

				s = string(b)
				if len(s) >= r.minSize && (r.maxSize == 0 || len(s) <= r.maxSize) {
					r.contents[s]++
					r.contentClusters[s] = append(r.contentClusters[s], newPathnameSet(pathname))
					r.contentPaths[s] = append(r.contentPaths[s], pathname)
				}
			}

			if newDir {
				r.numDirs++
				newDir = false
			}
			if len(s) >= r.minSize && (r.maxSize == 0 || len(s) <= r.maxSize) {
				r.pc[pathname] = s
				r.numFiles++
			}
		}
	}
	r.numDirs++ // Add in top level directory
	return r
}

func runAndCheckFileCounts(t *testing.T, opts Options, r *randTestVals) *Results {
	opts.MaxFileSize = uint64(r.maxSize)
	opts.MinFileSize = uint64(r.minSize)
	result, err := Run([]string{"."}, opts)
	if err != nil {
		t.Errorf("Error with Run() on random test files: %v", err)
	}

	if r.numDirs != result.DirCount {
		t.Errorf("Expected %v dirs, got: %v", r.numDirs, result.DirCount)
	}
	if r.numFiles != result.FileCount {
		t.Errorf("Expected %v files, got: %v", r.numFiles, result.FileCount)
	}
	return &result
}

func checkRunStats(t *testing.T, r *randTestVals, result *Results) {
	// Count how many times file content was used more than once.  The
	// result should equal the number of LinkPaths (ie. sets of pathnames
	// to link together).
	numLinkPaths := 0
	for _, v := range r.contents {
		if v > 1 {
			numLinkPaths++
		}
	}
	if numLinkPaths != len(result.LinkPaths) {
		t.Errorf("Expected %v LinkPaths, got: %v", numLinkPaths, len(result.LinkPaths))
	}

	// Check to see if our expected NewLinkCount matches what was computed.
	// This is done by having kept track of "clusters" when setting up the
	// test files (ie. grouping files with equal content by keeping track
	// of those that are linked together before the Run()
	for co, cl := range r.contentClusters {
		r.numInodes += int64(len(cl))

		// Sort by highest cluster count to lowest
		sort.Slice(cl, func(i, j int) bool { return len(cl[i]) > len(cl[j]) })
		// Doesn't handle maxNlink scenarios
		for i, m := range cl {
			r.numNlinks += int64(len(m))

			// The first cluster (with highest nlink count) is skipped,
			// because they will be linked to, not from, so aren't counted
			// by the NewLinkCount
			if i > 0 {
				r.numNewLinks += int64(len(m))
				r.linkPathsBytes += uint64(len(co))
			}

			// Also count the prev links using the cluster information.
			// Clusters of more than 1 pathname are pre-existing.
			if len(m) > 1 {
				r.numExistingLinks += int64(len(m) - 1)
				r.existingLinksBytes += uint64(len(co) * (len(m) - 1))
			}
		}
	}
	if r.numInodes != result.InodeCount {
		t.Errorf("Expected %v inodes, got: %v",
			r.numInodes, result.InodeCount)
	}
	if r.numNlinks != result.NlinkCount {
		t.Errorf("Expected %v nlinks, got: %v",
			r.numNlinks, result.NlinkCount)
	}
	if r.numNewLinks != result.NewLinkCount {
		t.Errorf("Expected %v NewLinkCount, got: %v",
			r.numNewLinks, result.NewLinkCount)
	}
	if r.numExistingLinks != result.ExistingLinkCount {
		t.Errorf("Expected %v ExistingLinkCount, got: %v",
			r.numExistingLinks, result.ExistingLinkCount)
	}
	if r.linkPathsBytes != result.InodeRemovedByteAmount {
		t.Errorf("Expected %v InodeRemovedByteAmount, got: %v",
			r.linkPathsBytes, result.InodeRemovedByteAmount)
	}
	if r.existingLinksBytes != result.ExistingLinkByteAmount {
		t.Errorf("Expected %v ExistingLinkedByteAmount, got: %v",
			r.existingLinksBytes, result.ExistingLinkByteAmount)
	}
}

type FilenameCounts map[string]int

func checkSameNameRunStats(t *testing.T, r *randTestVals, result *Results) {
	// Verify same filename is used in all LinkPaths
	for _, paths := range result.LinkPaths {
		filenames := map[string]struct{}{}
		for _, p := range paths {
			filenames[path.Base(p)] = struct{}{}
		}
		if len(filenames) > 1 {
			t.Errorf("SameName LinkPaths has mismatched filenames: %+v", result.LinkPaths)
		}
	}

	// Count the number of new links by counting the number of pathnames
	// with matching filenames, but *not* counting those used in the first
	// link cluster where the filename is encountered.
	for _, clusters := range r.contentClusters {
		// Sort by max nlink to min to replicate algorithm.  The
		// results will differ otherwise
		sort.Slice(clusters, func(i, j int) bool { return len(clusters[i]) > len(clusters[j]) })

		// The number of filenames to be linked to (ie. the number found after
		// the initial cluster)
		linkableFC := FilenameCounts{}

		// Cluster index where a filename was first encountered
		firstSeen := FilenameCounts{}

		// For each cluster in the list of clusters, keep track of which one
		// holds the first appearance of a filename (of any given pathname),
		// and don't count the filename (or names) in the first encountered
		// cluster (which would act as the src inode, not the destination).
		for i, cluster := range clusters {
			for pathname := range cluster {
				filename := path.Base(pathname)
				whenSeen, ok := firstSeen[filename]
				if !ok {
					firstSeen[filename] = i
				} else if whenSeen < i {
					linkableFC[filename]++
				}
			}
		}
		for _, count := range linkableFC {
			r.numNewLinks += int64(count)
		}
	}
	if r.numNewLinks != result.NewLinkCount {
		t.Errorf("Expected %v NewLinkCount, got: %v\n", r.numNewLinks, result.NewLinkCount)
	}
}

// TestRandFiles creates a bunch of files with random content, some with equal
// contents, and some pre-linked.  It checks that the result of a linking run
// are as expected.
func TestRandFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RandFiles test in short mode")
	}

	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, ContentOnly)
	r := setupRandTestFiles(t, topdir, opts.SameName)
	results := runAndCheckFileCounts(t, opts, r)
	checkRunStats(t, r, results)
}

func TestRandSameNameFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RandFiles test in short mode")
	}

	topdir := setUp("Run", t)
	defer os.RemoveAll(topdir)

	opts := SetupOptions(LinkingEnabled, ContentOnly, SameName)
	r := setupRandTestFiles(t, topdir, opts.SameName)
	results := runAndCheckFileCounts(t, opts, r)
	checkSameNameRunStats(t, r, results)
}
