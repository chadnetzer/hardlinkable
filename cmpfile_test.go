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
	"strings"
	"testing"
)

func TestFileContentComparison(t *testing.T) {
	R := strings.NewReader
	s := status{}
	s.Results = &Results{}
	s.Progress = &disabledProgress{}

	eq, err := readerContentsEqual(s, R(""), R(""))
	if !eq || err != nil {
		t.Errorf("Zero length cmpContents() compared unequal")
	}

	eq, err = readerContentsEqual(s, R("1"), R("1"))
	if !eq || err != nil {
		t.Errorf("Equal length 1 cmpContents() compared unequal")
	}

	eq, err = readerContentsEqual(s, R("1234"), R("123"))
	if eq || err != nil {
		t.Errorf("Unequal length cmpContents() compared equal")
	}

	eq, err = readerContentsEqual(s, R("A"), R("B"))
	if eq || err != nil {
		t.Errorf("Unequal length 1 cmpContents() compared equal")
	}
}
