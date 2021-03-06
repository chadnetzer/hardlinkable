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

package inode

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

type InoStatInfo map[Ino]*StatInfo

// os.FileInfo and syscall.Stat_t fields that we care about
type StatInfo struct {
	Size  uint64
	Ino   Ino
	Nlink uint64
	Uid   uint32
	Gid   uint32
	Mode  os.FileMode
	Mtim  time.Time
}

// We need the Dev value returned from stat, but it can be discarded when we
// separate the Info into a map indexed by the Dev value
type DevStatInfo struct {
	Dev uint64
	StatInfo
}

func LStatInfo(pathname string) (DevStatInfo, error) {
	fi, err := os.Lstat(pathname)
	if err != nil {
		return DevStatInfo{}, err
	}
	stat_t, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		errString := fmt.Errorf("Couldn't convert Stat_t for pathname: %s", pathname)
		return DevStatInfo{}, errString
	}
	di := DevStatInfo{
		Dev: uint64(stat_t.Dev),
		StatInfo: StatInfo{
			Size:  uint64(stat_t.Size),
			Ino:   Ino(stat_t.Ino),
			Nlink: uint64(stat_t.Nlink),
			Uid:   uint32(stat_t.Uid),
			Gid:   uint32(stat_t.Gid),
			Mode:  fi.Mode(),
			Mtim:  fi.ModTime(),
		},
	}

	return di, nil
}
