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
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/xattr"
)

func TestEqualXattrs(t *testing.T) {
	useDefer := true

	dir, err := ioutil.TempDir("", "hardlinkable")
	if err != nil {
		t.Fatalf("Couldn't create temp dir for equal Xattr tests: %v", err)
	}
	if useDefer {
		defer os.RemoveAll(dir)
	}

	if os.Chdir(dir) != nil {
		t.Fatalf("Couldn't chdir to temp dir for equal Xattr tests")
	}

	f1, err := ioutil.TempFile(dir, "f1")
	if err != nil {
		t.Fatalf("Couldn't create temp file for equal Xattr tests: %v", err)
	}
	if useDefer {
		defer os.Remove(f1.Name())
	}
	f2, err := ioutil.TempFile(dir, "f2")
	if err != nil {
		t.Fatalf("Couldn't create temp file for equal Xattr tests: %v", err)
	}
	if useDefer {
		defer os.Remove(f2.Name())
	}

	if eq, err := equalXAttrs(f1.Name(), f2.Name()); !eq || err != nil {
		t.Errorf("Unexpected Xattr mismatch for files %s and %s.  Should have no attributes: %v", f1.Name(), f2.Name(), err)
	}

	err = xattr.LSet(f1.Name(), "a", []byte("a1"))
	if err != nil {
		t.Fatalf("Couldn't LSet key 'a' to 'a1' on file1 %v: %v", f1, err)
	}

	if eq, err := equalXAttrs(f1.Name(), f2.Name()); eq || err != nil {
		t.Errorf("Unexpected Xattr match or error for files %s and %s.: %v", f1.Name(), f2.Name(), err)
	}

	err = xattr.LSet(f2.Name(), "a", []byte("a1"))
	if err != nil {
		t.Fatalf("Couldn't LSet key 'a' to 'a1' on file2 %v: %v", f1, err)
	}

	if eq, err := equalXAttrs(f1.Name(), f2.Name()); !eq || err != nil {
		t.Errorf("Unexpected Xattr mismatch or error for files %s and %s.: %v", f1.Name(), f2.Name(), err)
	}

	err = xattr.LSet(f1.Name(), "b", []byte("b1"))
	if err != nil {
		t.Fatalf("Couldn't LSet key 'b' to 'b1' on file %v: %v", f1, err)
	}

	if eq, err := equalXAttrs(f1.Name(), f2.Name()); eq || err != nil {
		t.Errorf("Unexpected Xattr match or error for files %s and %s.: %v", f1.Name(), f2.Name(), err)
	}
}
