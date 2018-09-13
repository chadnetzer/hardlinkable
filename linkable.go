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

package main

import (
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Linkable struct {
	FSDevs   map[uint64]FSDev
	options  *Options
	stats    *LinkingStats
	progress Progress
}

var MyLinkable *Linkable

func NewLinkable() *Linkable {
	var l Linkable
	l.FSDevs = make(map[uint64]FSDev)
	return &l
}

func init() {
	MyLinkable = NewLinkable()
}

func (ln *Linkable) Dev(dsi DevStatInfo, pathname string) FSDev {
	if fsdev, ok := ln.FSDevs[dsi.Dev]; ok {
		return fsdev
	} else {
		fsdev = NewFSDev(ln.options, ln.stats, dsi.Dev, MaxNlink(pathname))
		ln.FSDevs[dsi.Dev] = fsdev
		return fsdev
	}
}

func cliRun(dirs []string, files []string, co cliOptions) {
	options := co.toOptions()
	Run(dirs, files, options)
}

func Run(dirs []string, files []string, options Options) {
	MyLinkable.options = &options
	MyLinkable.stats = newLinkingStats(&options)

	if options.ProgressOutputEnabled &&
		terminal.IsTerminal(int(os.Stdout.Fd())) {
		MyLinkable.progress = NewTTYProgress(MyLinkable.stats, &options)
	} else {
		MyLinkable.progress = &DisabledProgress{}
	}
	MyLinkable.stats.StartTime = time.Now()
	c := MyLinkable.stats.MatchedPathnames(dirs, files, options)
	for pathname := range c {
		MyLinkable.progress.Show()
		dsi, err := LStatInfo(pathname)
		if err != nil {
			continue
		}
		if dsi.Size < options.MinFileSize {
			MyLinkable.stats.foundFileTooSmall()
			continue
		}
		if options.MaxFileSize > 0 &&
			dsi.Size > options.MaxFileSize {
			MyLinkable.stats.foundFileTooLarge()
			continue
		}
		// If the file hasn't been rejected by this
		// point, add it to the found count
		MyLinkable.stats.foundFile()

		fsdev := MyLinkable.Dev(dsi, pathname)
		fsdev.findIdenticalFiles(dsi, pathname)
	}

	MyLinkable.progress.Clear()

	// Calculate and store the number of unique paths and directories
	// encountered by the walk, overwriting the less accurate counts
	// gathered during the walk.
	var numPaths, numDirs int64
	for _, fsdev := range MyLinkable.FSDevs {
		p, d := fsdev.pathCount()
		numPaths += p
		numDirs += d
	}
	MyLinkable.stats.fileAndDirectoryCount(numPaths, numDirs)

	// Iterate over all the inode sorted links.  We discard each link pair
	// (for now), since the links are stored in the Stats type.
	for _, fsdev := range MyLinkable.FSDevs {
		for pair := range fsdev.sortedLinks() {
			_ = pair
		}
	}
	MyLinkable.stats.EndTime = time.Now()
	if options.JSONOutputEnabled {
		MyLinkable.stats.outputJSONResults()
	} else {
		MyLinkable.stats.outputResults()
	}
}
