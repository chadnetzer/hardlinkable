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
func MatchedPathnames(directories []string) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, dir := range directories {
			err := godirwalk.Walk(dir, &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					if de.ModeType().IsDir() {
						// Do not exclude dirs provided explicitly by the user
						if dir != osPathname &&
							isExcluded(de.Name(), MyOptions.DirExcludes) {
							return filepath.SkipDir
						}
						Stats.FoundDirectory()
					} else if de.ModeType().IsRegular() {
						if !isExcluded(de.Name(), MyOptions.FileExcludes) {
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

func isExcluded(name string, pattern []string) bool {
	for _, p := range pattern {
		matched, err := regexp.MatchString(p, name)
		if matched && err == nil {
			return true
		}
	}
	return false
}
