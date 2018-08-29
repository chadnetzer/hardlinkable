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
	"bytes"

	"github.com/pkg/xattr"
)

func equalXAttrs(pathname1, pathname2 string) (bool, error) {
	var list1, list2 []string
	var err error
	if list1, err = xattr.LList(pathname1); err != nil {
		return false, err
	}

	if list2, err = xattr.LList(pathname2); err != nil {
		return false, err
	}

	if len(list1) != len(list2) {
		return false, nil
	}

	// Make list1 the longer list, and make it and it's values into a map
	if len(list1) < len(list2) {
		list1, list2 = list2, list1
		pathname1, pathname2 = pathname2, pathname1
	}

	d := make(map[string][]byte, len(list1))
	for _, key := range list1 {
		d[key], err = xattr.LGet(pathname1, key)
		if err != nil {
			return false, err
		}
	}

	for _, key := range list2 {
		v1, ok := d[key]
		if !ok {
			return false, nil
		}
		v2, err := xattr.LGet(pathname2, key)
		if err != nil {
			return false, nil
		}
		if bytes.Compare(v1, v2) != 0 {
			return false, nil
		}
	}

	return true, nil
}
