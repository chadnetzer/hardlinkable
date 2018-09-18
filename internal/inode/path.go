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
	P "hardlinkable/internal/pathpool"
)

type PathInfo struct {
	P.Pathsplit
	StatInfo
}

type PathInfoPair struct {
	Src PathInfo
	Dst PathInfo
}

func (p1 PathInfo) EqualTime(p2 PathInfo) bool {
	return p1.Sec == p2.Sec && p1.Nsec == p2.Nsec
}

func (p1 PathInfo) EqualMode(p2 PathInfo) bool {
	return p1.Mode.Perm() == p2.Mode.Perm()
}

func (p1 PathInfo) EqualOwnership(p2 PathInfo) bool {
	return p1.Uid == p2.Uid && p1.Gid == p2.Gid
}
