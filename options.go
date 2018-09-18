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

const DefaultSearchThresh = 1
const DefaultMinFileSize = 1
const DefaultStoreExistingLinkResults = true // Non-cli default
const DefaultStoreNewLinkResults = true      // Non-cli default
const DefaultShowExtendedRunStats = false    // Non-cli default

// Options is passed to the Run() func, and controls the operation of the
// hardlinkable algorithm, including what inode parameters much match for files
// to be compared for equality, what files and directories are included or
// excluded, and whether linking is actually enabled or not.
type Options struct {
	// SameName enabled ensures only files with matching filenames can be
	// linked
	SameName bool

	// IgnoreTime enabled allows files with different mtime values can be
	// linked
	IgnoreTime bool

	// IgnorePerms enabled allows files with different inode mode values
	// can be linked
	IgnorePerms bool

	// IgnoreOwner enabled allows files with different uid or gid can be
	// linked
	IgnoreOwner bool

	// IgnoreXattr enabled allows files with different xattrs can be linked
	IgnoreXattr bool

	// LinkingEnabled causes the Run to perform the linking step
	LinkingEnabled bool

	// Verbosity controls the level of output when calling the output
	// options.  Verbosity 0 prints a short summary of results (space
	// saved, etc.). Verbosity 1 outputs additional information on
	// comparison results and other stats.  Verbosity 2 also outputs the
	// linking that would be (or was) performed, and Verbosity 3 prints
	// information on what existing hardlinks were encountered.
	Verbosity int

	// DebugLevel controls the amount of debug information reported in the
	// results output, as well as debug logging.
	DebugLevel int

	// SearchThresh determines the length that the lists of files with
	// equivalent inode hashes can grow to, before also enabling content
	// digests (which can drastically reduce the number of compared files
	// when there are many with the same hash, but differing content at the
	// start of the file).  Can be disabled with -1.  May save a small
	// amount of memory, but potentially at greatly increased runtime in
	// worst case scenarios with many, many files.
	SearchThresh int

	// MinFileSize controls the minimum size of files that are eligible to
	// be considered for linking.
	MinFileSize uint64

	// MaxFileSize controls the maximum size of files that are eligible to
	// be considered for linking.
	MaxFileSize uint64

	// FileIncludes is a slice of regex expressions that control what
	// filenames will be considered for linking.  If given without any
	// FileExcludes, the walked files must match one of the includes.  If
	// FileExcludes are provided, the FileIncludes can override them.
	FileIncludes []string

	// FileExcludes is a slice of regex expressions that control what
	// filenames will be excluded from consideration for linking.
	FileExcludes []string

	// DirExcludes is a slice of regex expressions that control what
	// directories will be excluded from the file discovery walk.
	DirExcludes []string

	// StoreExistingLinkResults allows controlling whether to store
	// discovered existing links in Results. Verbosity > 2 can override.
	StoreExistingLinkResults bool

	// StoreNewLinkResults allows controlling whether to store discovered
	// new hardlinkable pathnames in Results. Verbosity > 1 can override.
	StoreNewLinkResults bool

	// ShowExtendedRunStats enabled displays additional Result stats
	// output.  Verbosity > 0 can override.
	ShowExtendedRunStats bool
}

// DefaultOptions returns an Options struct, with the defaults initialized.
func DefaultOptions() Options {
	o := Options{
		SearchThresh:             DefaultSearchThresh,
		MinFileSize:              DefaultMinFileSize,
		StoreExistingLinkResults: DefaultStoreExistingLinkResults,
		StoreNewLinkResults:      DefaultStoreNewLinkResults,
		ShowExtendedRunStats:     DefaultShowExtendedRunStats,
	}
	return o
}
