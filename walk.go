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
	"log"
	"path/filepath"
	"regexp"

	"github.com/karrick/godirwalk"
)

// Return allowed pathnames through the given channel.
func matchedPathnames(opts Options, r *Results, dirs []string, files []string) <-chan string {
	// Options is a copy to prevent being changed during walk.
	out := make(chan string)
	go func() {
		defer close(out)
		for _, dir := range dirs {
			err := godirwalk.Walk(dir, &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					if de.ModeType().IsDir() {
						dirExcludes := opts.DirExcludes
						// Do not exclude dirs provided explicitly by the user
						if dir != osPathname && isMatched(de.Name(), dirExcludes) {
							return filepath.SkipDir
						}
					} else if de.ModeType().IsRegular() {
						if isFileIncluded(de.Name(), &opts) {
							out <- osPathname
						}
					}
					return nil
				},
			})
			if err != nil {
				log.Printf("Couldn't walk \"%v\" dir: %v. Skipping...", dir, err)
			}
		}
		// Also pass back some or all (depending on includes and
		// excludes) of the passed in file pathnames.
		for _, pathname := range files {
			if isFileIncluded(pathname, &opts) {
				out <- pathname
			}
		}
	}()
	return out
}

// isMatched() returns true if name matches any of the patterns, and false
// otherwise (or if there are no patterns).
func isMatched(name string, pattern []string) bool {
	for _, p := range pattern {
		matched, err := regexp.MatchString(p, name)
		if matched && err == nil {
			return true
		}
	}
	return false
}

// isFileIncluded returns true if the given pathname is not excluded, or is
// specifically included by the command line options.
func isFileIncluded(name string, opts *Options) bool {
	inc := opts.FileIncludes
	exc := opts.FileExcludes
	if len(exc) == 0 && len(inc) == 0 {
		return true
	}
	if len(inc) > 0 && isMatched(name, inc) {
		return true
	}
	if len(exc) > 0 && !isMatched(name, exc) {
		return true
	}
	return false
}
