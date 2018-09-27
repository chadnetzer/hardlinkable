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
	I "hardlinkable/internal/inode"
	P "hardlinkable/internal/pathpool"
	"io/ioutil"
	"os"
	"testing"
)

func TestDoLink(t *testing.T) {
	options := &Options{}
	results := newResults(options)
	ws := status{options, results, nil}
	fs := newFSDev(ws, 10000, 10000) // Arbitrary args
	topdir, err := ioutil.TempDir("", "hardlinkable")
	if err != nil {
		t.Fatalf("Couldn't create temp dir for doLink tests: %v", err)
	}
	defer os.RemoveAll(topdir)

	if os.Chdir(topdir) != nil {
		t.Fatalf("Couldn't chdir to temp dir for doLink tests")
	}

	f1, err := ioutil.TempFile(topdir, "f1")
	if err != nil {
		t.Fatalf("Couldn't create temp file for doLink tests: %v", err)
	}
	defer os.Remove(f1.Name())

	f2, err := ioutil.TempFile(topdir, "f2")
	if err != nil {
		t.Fatalf("Couldn't create temp file for doLink tests: %v", err)
	}
	defer os.Remove(f2.Name())

	dsi1, err := I.LInfo(f1.Name())
	if err != nil {
		t.Fatalf("Couldn't run LInfo(f1.Name()): %v", err)
	}

	dsi2, err := I.LInfo(f2.Name())
	if err != nil {
		t.Fatalf("Couldn't run LInfo(f2.Name()): %v", err)
	}

	if dsi1.Dev != dsi2.Dev {
		t.Fatalf("f1 and f2 device numbers do not match: %v %v", dsi1.Dev, dsi2.Dev)
	}

	fs.Dev = dsi1.Dev
	fs.InoStatInfo[dsi1.Ino] = &dsi1.StatInfo
	fs.InoStatInfo[dsi2.Ino] = &dsi2.StatInfo

	ps1 := I.PathInfo{P.Split(f1.Name(), nil), dsi1.StatInfo}
	ps2 := I.PathInfo{P.Split(f2.Name(), nil), dsi2.StatInfo}
	err = fs.hardlinkFiles(ps1, ps2)
	if err != nil {
		t.Errorf("Linking ps1 and ps2 failed: %v %v", dsi1, dsi2)
	}

	dsi11, err := I.LInfo(f1.Name())
	dsi12, err := I.LInfo(f1.Name())

	if dsi11 != dsi12 {
		t.Errorf("Linked path inodes are unequal: %+v %+v", dsi11, dsi12)
	}
	if dsi11.Nlink != 2 {
		t.Errorf("Linked path inode expeced nlink=2, got nlink=%v", dsi11.Nlink)
	}

	f3, err := ioutil.TempFile(topdir, "f3")
	if err != nil {
		t.Fatalf("Couldn't create temp file for doLink tests: %v", err)
	}
	defer os.Remove(f3.Name())

	dsi3, err := I.LInfo(f3.Name())
	if err != nil {
		t.Fatalf("Couldn't run LInfo(f3.Name()): %v", err)
	}
	// Deliberately create a mismatch between the file's stat info, and the
	// stored stat info
	dsi3.StatInfo.Sec -= 999
	fs.InoStatInfo[dsi3.Ino] = &dsi3.StatInfo
	ps3 := I.PathInfo{P.Split(f3.Name(), nil), dsi3.StatInfo}

	err = fs.hardlinkFiles(ps1, ps3)
	if err == nil {
		t.Errorf("Linking ps1 and ps3 was expected to fail: %+v %+v", dsi11, dsi3)
	}
}

func TestHasBeenModified(t *testing.T) {
	topdir, err := ioutil.TempDir("", "hardlinkable")
	if err != nil {
		t.Fatalf("Couldn't create temp dir for %v tests: %v", topdir, err)
	}
	defer os.RemoveAll(topdir)

	if os.Chdir(topdir) != nil {
		t.Fatalf("Couldn't chdir to temp dir for %v tests", topdir)
	}

	// Create single byte file
	filename := "f1"
	if err := ioutil.WriteFile(filename, []byte{'X'}, 0644); err != nil {
		t.Fatalf("Couldn't create test file '%v'", filename)
	}

	// Make PathInfo for created file
	dsi, err := I.LInfo(filename)
	if err != nil {
		t.Fatalf("Couldn't stat test file '%v'", filename)
	}
	p := P.Pathsplit{".", filename}
	pi := I.PathInfo{p, dsi.StatInfo}

	// Change Dev so that hasBeenModified() returns true
	if !hasBeenModified(pi, dsi.Dev+1) {
		t.Errorf("Failed to detect Dev modification to file: '%v'", filename)
	}

	// Change Ino on the PathInfo, so that hasBeenModified() returns true
	newPI := pi
	newPI.Ino += 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Ino modification to file: '%v'", filename)
	}

	// Change Nlink on the PathInfo, so that hasBeenModified() returns true
	newPI = pi
	newPI.Nlink += 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Nlink modification to file: '%v'", filename)
	}

	// Change PathInfo time, so that hasBeenModified() returns true
	newPI = pi
	newPI.Sec -= 86400
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Sec time modification to file: '%v'", filename)
	}
	newPI = pi
	newPI.Nsec = pi.Nsec*2 + 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Nsec time modification to file: '%v'", filename)
	}

	// Change PathInfo ownership, so that hasBeenModified() returns true
	newPI = pi
	newPI.Uid += 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect UID modification to file: '%v'", filename)
	}
	newPI = pi
	newPI.Gid += 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect GID modification to file: '%v'", filename)
	}

	// Change PathInfo ownership, so that hasBeenModified() returns true
	newPI = pi
	newPI.Mode ^= 1
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Mode modification to file: '%v'", filename)
	}

	// Change PathInfo Size, so that hasBeenModified() returns true
	newPI = pi
	newPI.Size *= 2
	if !hasBeenModified(newPI, dsi.Dev) {
		t.Errorf("Failed to detect Size modification to file: '%v'", filename)
	}
}
