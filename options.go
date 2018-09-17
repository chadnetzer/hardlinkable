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

// Options is passed to the Run() func, and controls the operation of the
// hardlinkable algorithm, including what inode parameters much match for files
// to be compared for equality, what files and directories are included or
// excluded, and whether linking is actually enabled or not.
type Options struct {
	StatsOutputEnabled    bool
	JSONOutputEnabled     bool
	SameName              bool
	IgnoreTime            bool
	IgnorePerms           bool
	IgnoreOwner           bool
	IgnoreXattr           bool
	LinkingEnabled        bool
	Verbosity             int
	DebugLevel            int
	SearchThresh          int
	MinFileSize           uint64
	MaxFileSize           uint64
	FileIncludes          []string
	FileExcludes          []string
	DirExcludes           []string

	// Indirect options, set based on debug and/or verbosity level
	existingLinkStatsEnabled bool
	newLinkStatsEnabled      bool
}

// DefaultOptions returns an Options struct, with the defaults initialized.
func DefaultOptions() Options {
	o := Options{
		SearchThresh: DefaultSearchThresh,
		MinFileSize:  DefaultMinFileSize,
	}
	return o
}

// init sets up the unexported Options, and must be called on an Options struct
// that has had it's exported members set to their desired values, (ie. on the
// Options provided by the user).
func (o *Options) init() *Options {
	if o.Verbosity > 1 {
		o.newLinkStatsEnabled = true
	}
	if o.Verbosity > 2 {
		o.existingLinkStatsEnabled = true
	}
	return o
}
