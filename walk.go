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
	"path/filepath"
	"regexp"

	"github.com/karrick/godirwalk"
)

// Return allowed pathnames through the given channel.
func MatchedPathnames(directories []string, options Options) <-chan string {
	seenDirs := make(map[string]struct{})
	out := make(chan string)
	go func() {
		defer close(out)
		for _, dir := range directories {
			if _, ok := seenDirs[dir]; ok {
				continue
			} else {
				seenDirs[dir] = struct{}{}
			}
			err := godirwalk.Walk(dir, &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					if de.ModeType().IsDir() {
						dirname := de.Name()
						dirExcludes := options.DirExcludes
						// Do not exclude dirs provided explicitly by the user
						if dir != osPathname && isMatched(dirname, dirExcludes) {
							return filepath.SkipDir
						}
						Stats.FoundDirectory()
					} else if de.ModeType().IsRegular() {
						filename := de.Name()
						fileIncludes := options.FileIncludes
						fileExcludes := options.FileExcludes
						// When excludes is not empty, include can override an exclude
						if (len(fileExcludes) == 0 && len(fileIncludes) == 0) ||
							(len(fileIncludes) > 0 && isMatched(filename, fileIncludes)) ||
							(len(fileExcludes) > 0 && !isMatched(filename, fileExcludes)) {
							out <- osPathname
						}
					}
					return nil
				},
			})
			if err != nil {
				fmt.Println(err)
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
