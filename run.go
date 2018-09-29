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
	"fmt"
	"hardlinkable/internal/inode"
	"log"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// RunWithProgress performs a scan of the supplied directories and files, with
// the given Options, and outputs information on which files could be linked to
// save space.  If stdout is a terminal/tty, a progress line is continually
// updated as the directories and files are scanned.
func RunWithProgress(dirs []string, files []string, opts Options) (Results, error) {
	var ls *linkableState = newLinkableState(&opts)

	if err := validateOptions(opts); err != nil {
		return *ls.Results, err
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		ls.Progress = newTTYProgress(ls.Results, ls.Options)
	} else {
		ls.Progress = &disabledProgress{}
	}

	err := runHelper(dirs, files, ls)
	return *ls.Results, err
}

// Run performs a scan of the supplied directories and files, with the given
// Options, and outputs information on which files could be linked to save
// space.
func Run(dirs []string, files []string, opts Options) (Results, error) {
	var ls *linkableState = newLinkableState(&opts)

	if err := validateOptions(opts); err != nil {
		return *ls.Results, err
	}

	ls.Progress = &disabledProgress{}

	err := runHelper(dirs, files, ls)
	return *ls.Results, err
}

// runHelper is called by the public Run funcs, with an already initialized
// options, to complete the scanning and result gathering.
func runHelper(dirs []string, files []string, ls *linkableState) (err error) {
	ls.Results.start()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Run stopped early: %v ", r)
		}
	}()
	defer ls.Results.end()
	defer ls.Progress.Done()

	c := matchedPathnames(*ls.Options, ls.Results, dirs, files)
	for pathname := range c {
		ls.Progress.Show()
		di, err := inode.LStatInfo(pathname)
		if err != nil {
			log.Printf("Couldn't stat(\"%v\"). Skipping...", pathname)
			continue
		}
		if di.Size < ls.Options.MinFileSize {
			ls.Results.foundFileTooSmall()
			continue
		}
		if ls.Options.MaxFileSize > 0 &&
			di.Size > ls.Options.MaxFileSize {
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
		p, d := fsdev.InoPaths.PathCount()
		numPaths += p
		numDirs += d
	}
	ls.Results.fileAndDirectoryCount(numPaths, numDirs)

	// Iterate over all the inode sorted links, to gather accurate linking
	// statistics, and optionally link them if requested.
	for _, fsdev := range ls.fsDevs {
		if err := fsdev.generateLinks(); err != nil {
			return err
		}
	}
	ls.Results.runCompletedSuccessfully()

	return nil
}

func validateOptions(opts Options) error {
	if opts.MaxFileSize > 0 && opts.MaxFileSize < opts.MinFileSize {
		return fmt.Errorf("minFileSize (%v) cannot be larger than maxFileSize (%v)",
			opts.MinFileSize, opts.MaxFileSize)
	}
	return nil
}
