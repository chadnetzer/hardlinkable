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
	"fmt"
	"strings"
	"time"
)

const numFPSes = 8

// A simple progress meter while scanning directories and performing linking
type Progress struct {
	lastLineLen     int
	lastTime        time.Time
	updateDelay     time.Duration
	dirFilesCounter int
	counterMin      int
	fpsHist         [numFPSes]float64
	fpsI            int
	lastAvgFPS      float64

	stats   *LinkingStats
	options *Options
}

func NewProgress(stats *LinkingStats, options *Options) Progress {
	p := Progress{
		updateDelay: 100 * time.Millisecond,
		counterMin:  11, // Prime number makes output more dynamic
		stats:       stats,
		options:     options,
	}
	return p
}

// Output a line (without a newline at the end) of progress on directory
// scanning and inode linking (ie. which inodes have identical content and
// matching inode parameters).  Call in the main directory walk/link
// calculation loop.
func (p *Progress) ShowDirsFilesFound() {
	if p.options.ProgressOutputEnabled {
		return
	}
	p.dirFilesCounter += 1
	if p.dirFilesCounter < p.counterMin {
		return
	} else {
		p.dirFilesCounter = 0
	}

	now := time.Now()
	timeSinceLast := now.Sub(p.lastTime)
	if timeSinceLast < p.updateDelay {
		return
	}
	p.lastTime = time.Now()

	numDirs := p.stats.numDirs
	numFiles := p.stats.numFiles

	duration := time.Now().Sub(p.stats.startTime)
	durStr := duration.Round(time.Second).String()
	fps := float64(numFiles) / duration.Seconds()

	// Calculate a simple windowed average of the FPS, just to help
	// determine roughly if the FPS rate is increasing or decreasing.  It's
	// deliberately simplistic, as it's used more as a direction flag.
	p.fpsHist[p.fpsI] = fps
	p.fpsI = (p.fpsI + 1) % numFPSes
	var fpsSum float64
	var directionStr string
	for _, v := range p.fpsHist {
		fpsSum += v
	}
	avgFPS := fpsSum / float64(numFPSes)
	if avgFPS > p.lastAvgFPS {
		directionStr = "⬇"
	} else {
		directionStr = "⬆"
	}
	p.lastAvgFPS = avgFPS

	fmtStr := "\r%d files in %d dirs (elapsed time: %s files/sec: %.0f %v)"
	s := fmt.Sprintf(fmtStr, numFiles, numDirs, durStr, fps, directionStr)
	p.line(s)
}

// Call to erase the progress loop (before 'normal' program post-processing
// output)
func (p *Progress) Clear() {
	p.line("\r")
	p.lastLineLen = 1
	p.line("\r")
}

func (p *Progress) line(s string) {
	numSpaces := p.lastLineLen - len(s)
	if numSpaces > 0 {
		s += strings.Repeat(" ", numSpaces)
	}
	p.lastLineLen = len(s)
	fmt.Print(s)
}
