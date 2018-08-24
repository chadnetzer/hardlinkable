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
	"time"
)

type Linkable struct {
	FSDevs map[uint64]FSDev
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
		fsdev = NewFSDev(dsi.Dev, MaxNlink(pathname))
		ln.FSDevs[dsi.Dev] = fsdev
		return fsdev
	}
}

func Run(dirs []string) {
	options := MyCLIOptions.NewOptions()
	MyOptions = &options // Compatibility setup for now

	Stats.startTime = time.Now()
	c := MatchedPathnames(dirs)
	for pathname := range c {
		dsi, err := LStatInfo(pathname)
		if err != nil {
			continue
		}
		if dsi.Size < options.MinFileSize {
			Stats.FoundFileTooSmall()
			continue
		}
		if options.MaxFileSize > 0 &&
			dsi.Size > options.MaxFileSize {
			Stats.FoundFileTooLarge()
			continue
		}
		// If the file hasn't been rejected by this
		// point, add it to the found count
		Stats.FoundFile()

		fsdev := MyLinkable.Dev(dsi, pathname)
		fsdev.findIdenticalFiles(dsi, pathname)
	}

	for _, fsdev := range MyLinkable.FSDevs {
		for pair := range fsdev.sortedLinks() {
			_ = pair
		}
	}
	Stats.endTime = time.Now()
	Stats.outputResults()
}
