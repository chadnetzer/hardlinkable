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
	"path"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// RunWithProgress performs a scan of the supplied directories and files, with
// the given Options, and outputs information on which files could be linked to
// save space.  If stdout is a terminal/tty, a progress line is continually
// updated as the directories and files are scanned.
func RunWithProgress(dirsAndFiles []string, opts Options) (Results, error) {
	var ls *linkableState = newLinkableState(&opts)

	var err error
	if err = opts.Validate(); err != nil {
		return *ls.Results, err
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		ls.Progress = newTTYProgress(ls.Results, ls.Options)
	} else {
		ls.Progress = &disabledProgress{}
	}

	err = runHelper(dirsAndFiles, ls)
	return *ls.Results, err
}

// Run performs a scan of the supplied directories and files, with the given
// Options, and outputs information on which files could be linked to save
// space.
func Run(dirsAndFiles []string, opts Options) (Results, error) {
	var ls *linkableState = newLinkableState(&opts)

	if err := opts.Validate(); err != nil {
		return *ls.Results, err
	}

	ls.Progress = &disabledProgress{}

	err := runHelper(dirsAndFiles, ls)
	return *ls.Results, err
}

// runHelper is called by the public Run funcs, with an already initialized
// options, to complete the scanning and result gathering.
func runHelper(dirsAndFiles []string, ls *linkableState) (err error) {
	dirs, files, err := ValidateDirsAndFiles(dirsAndFiles)
	if err != nil {
		return err
	}

	ls.Results.start()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Run stopped early: %v ", r)
		}
	}()
	defer ls.Results.end()
	defer ls.Progress.Done()

	// Phase 1: Gather path and inode information by walking the dirs and
	// files, looking for files that can be linked due to identical
	// contents, and optionally equivalent inode parameters (time,
	// permission, ownership, etc.)
	ls.Results.Phase = WalkPhase
	c := matchedPathnames(*ls.Options, ls.Results, dirs, files)
	for pe := range c {
		// Handle early termination of the directory walk.  If
		// IgnoreWalkErrors is set, we won't get any errors here.
		if pe.err != nil {
			return pe.err
		}

		ls.Progress.Show()
		di, statErr := inode.LStatInfo(pe.pathname)
		if statErr != nil {
			if ls.Options.IgnoreWalkErrors {
				ls.Results.SkippedFileErrCount++
				if ls.Options.DebugLevel > 0 {
					log.Printf("\r%v  Skipping...", statErr)
				}
				continue
			} else {
				return statErr
			}
		}

		// Ignore files with setuid/setgid bits.  Linking them could
		// have security implications.
		if di.Mode&os.ModeSetuid != 0 {
			ls.Results.foundSetuidFile()
			continue
		}
		if di.Mode&os.ModeSetgid != 0 {
			ls.Results.foundSetgidFile()
			continue
		}

		// Also exclude files with any other non-perm mode bits set
		if di.Mode != (di.Mode & os.ModePerm) {
			ls.Results.foundNonPermBitFile()
			continue
		}

		// Ensure the files fall within the allowed Size range
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

		fsdev := ls.dev(di, pe.pathname)
		cmpErr := fsdev.FindIdenticalFiles(di, pe.pathname)
		if cmpErr != nil {
			if ls.Options.IgnoreWalkErrors {
				ls.Results.SkippedFileErrCount++
				if ls.Options.DebugLevel > 0 {
					log.Printf("\r%v  Skipping...", cmpErr)
				}
			} else {
				return cmpErr
			}
		}
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

	// Phase 2: Link generation - with all the path and inode information
	// collected, iterate over all the inode links sorted from highest
	// nlink count to lowest, gathering accurate linking statistics,
	// determine what link() pairs and in what order are needed to produce
	// the desired result, and optionally link them if requested.
	ls.Results.Phase = LinkPhase
	for _, fsdev := range ls.fsDevs {
		if err := fsdev.generateLinks(); err != nil {
			return err
		}
	}
	ls.Results.runCompletedSuccessfully()

	return nil
}

type devIno struct {
	dev uint64
	ino uint64
}

// ValidateDirs will ensure only dirs are provided, and remove duplicates.  It
// is called by Run() to check the 'dirs' arg.
func ValidateDirsAndFiles(dirsAndFiles []string) (dirs []string, files []string, err error) {
	dirs = []string{}
	files = []string{}
	seenDirs := make(map[devIno]struct{})
	seenFiles := make(map[string]struct{})
	for _, name := range dirsAndFiles {
		var fi os.FileInfo
		fi, err = os.Lstat(name)
		if err != nil {
			return
		}
		if fi.IsDir() {
			stat_t, ok := fi.Sys().(*syscall.Stat_t)
			if !ok {
				err = fmt.Errorf("Couldn't convert Stat_t for pathname: %s", name)
				return
			}
			di := devIno{dev: uint64(stat_t.Dev), ino: uint64(stat_t.Ino)}
			if _, ok := seenDirs[di]; ok {
				continue
			}
			seenDirs[di] = struct{}{}
			dirs = append(dirs, name)
			continue
		}
		if fi.Mode().IsRegular() {
			name = path.Clean(name)
			if _, ok := seenFiles[name]; ok {
				continue
			}
			seenFiles[name] = struct{}{}
			files = append(files, name)
			continue
		}

		err = fmt.Errorf("'%v' is not a directory or a 'regular' file", name)
		return
	}
	return
}
