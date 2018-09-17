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

// hardlinkable determines which files in the given directories have equal
// content and compatible inode properties, and returns information on the
// space that would be saved by hardlinking them all together.  It can also,
// optionally, perform the hardlinking.
package hardlinkable

import (
	"hardlinkable/internal/inode"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// Run performs a scan of the supplied directories and files, with the given
// Options, and outputs information on which files could be linked to save
// space.
func Run(dirs []string, files []string, opts Options) Results {
	var ls *linkableState = newLinkableState()

	ls.Options = opts.init()
	ls.Results = newResults(ls.Options)

	if ls.Options.ProgressOutputEnabled &&
		terminal.IsTerminal(int(os.Stdout.Fd())) {
		ls.Progress = newTTYProgress(ls.Results, ls.Options)
	} else {
		ls.Progress = &disabledProgress{}
	}

	ls.Results.start()
	c := matchedPathnames(ls.status, dirs, files)
	for pathname := range c {
		ls.Progress.Show()
		di, err := inode.LInfo(pathname)
		if err != nil {
			continue
		}
		if di.Size < opts.MinFileSize {
			ls.Results.foundFileTooSmall()
			continue
		}
		if opts.MaxFileSize > 0 &&
			di.Size > opts.MaxFileSize {
			ls.Results.foundFileTooLarge()
			continue
		}
		// If the file hasn't been rejected by this
		// point, add it to the found count
		ls.Results.foundFile()

		fsdev := ls.dev(di, pathname)
		fsdev.FindIdenticalFiles(di, pathname)
	}

	ls.Progress.Clear()

	// Calculate and store the number of unique paths and directories
	// encountered by the walk, overwriting the less accurate counts
	// gathered during the walk.
	var numPaths, numDirs int64
	for _, fsdev := range ls.fsDevs {
		p, d := fsdev.PathCount()
		numPaths += p
		numDirs += d
	}
	ls.Results.fileAndDirectoryCount(numPaths, numDirs)

	// Iterate over all the inode sorted links.  We discard each link pair
	// (for now), since the links are stored in the Results type.
	for _, fsdev := range ls.fsDevs {
		for pair := range fsdev.SortedLinks() {
			_ = pair
		}
	}
	ls.Results.end()

	return *ls.Results
}
