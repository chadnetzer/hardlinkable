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
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type progress interface {
	Show()
	Clear()
	Done()
}

// A simple progress meter while scanning directories and performing linking
type ttyProgress struct {
	lastLineLen    int
	lastFPSTime    time.Time
	updateDelay    time.Duration
	updateFPSDelay time.Duration
	lastFPS        float64
	bytesCompared  uint64

	timer     chan struct{}
	done      chan struct{}
	showsOver bool

	results *Results
	options *Options

	m runtime.MemStats
}

type disabledProgress struct{}

// Initialize TTYProgress and return pointer to it
func newTTYProgress(results *Results, options *Options) *ttyProgress {
	if options.LinkingEnabled {
		fmt.Println(" -- Linking enabled. Filesystem will be modified. CTRL-C to abort --")
	}
	now := time.Now()
	p := ttyProgress{
		lastFPSTime:    now,
		updateDelay:    60 * time.Millisecond,
		updateFPSDelay: 180 * time.Millisecond, // A slower rate for readability
		timer:          make(chan struct{}),
		done:           make(chan struct{}),
		results:        results,
		options:        options,
	}

	// Send a message after delaying, controlling progress update rate
	go func() {
		defer close(p.timer)
		for {
			time.Sleep(p.updateDelay)
			select {
			case <-p.done:
				return
			default:
				p.timer <- struct{}{}
			}
		}
	}()

	return &p
}

// Output a line (without a newline at the end) of progress on directory
// scanning and inode linking (ie. which inodes have identical content and
// matching inode parameters).  Call in the main directory walk/link
// calculation loop.
func (p *ttyProgress) Show() {
	// Return if our timer hasn't yet fired
	select {
	case <-p.timer:
		// updateDelay elapsed.  Continue func...
	default:
		return
	}

	// No more timer heartbeats will be coming
	if p.showsOver {
		return
	}

	now := time.Now()

	numFiles := p.results.FileCount

	duration := now.Sub(p.results.StartTime)
	durStr := duration.Round(time.Second).String()

	var fps float64
	timeSinceLastFPS := now.Sub(p.lastFPSTime)
	if timeSinceLastFPS > p.updateFPSDelay {
		fps = float64(numFiles) / duration.Seconds()
		p.lastFPS = fps
		p.lastFPSTime = now

		p.bytesCompared = p.results.BytesCompared

		if p.options.DebugLevel > 1 {
			runtime.ReadMemStats(&p.m)
		}
	} else {
		fps = p.lastFPS
	}

	decimals := 3
	if p.bytesCompared > 1024*1024*1024*1024 { // 1 TiB
		decimals = 6
	}

	fmtStr := "\r%d files in %s (%.0f/sec)  compared %v"
	s := fmt.Sprintf(fmtStr, numFiles, durStr, fps,
		HumanizeWithPrecision(p.bytesCompared, decimals))

	if p.options.DebugLevel > 1 {
		s += fmt.Sprintf("  Allocs %v", Humanize(p.m.Alloc))
	}
	p.line(s)
}

// Call to erase the progress loop (before 'normal' program post-processing
// output)
func (p *ttyProgress) Clear() {
	p.line("\r")
	p.lastLineLen = 0
	p.line("\r")
}

// Done indicates that the use of the progress interface is over
func (p *ttyProgress) Done() {
	if !p.showsOver {
		p.showsOver = true
		close(p.done) // done
	}
	p.Clear()
}

// line outputs a string that is right-padded with enough space to overwrite
// the previous line
func (p *ttyProgress) line(s string) {
	thisLen := len(s) - 1 // Ignore the '\r'
	numSpaces := p.lastLineLen - thisLen
	if numSpaces > 0 {
		s += strings.Repeat(" ", numSpaces)
	}
	width, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err == nil && width < thisLen {
		s = s[:width]
	}

	fmt.Print(s)
	p.lastLineLen = thisLen
}

func (p *disabledProgress) Show()  {}
func (p *disabledProgress) Clear() {}
func (p *disabledProgress) Done()  {}
