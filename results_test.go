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
	"testing"
)

func TestHumanize(t *testing.T) {
	h := map[uint64]string{
		0:                                    "0 bytes",
		1:                                    "1 bytes",
		1023:                                 "1023 bytes",
		1024:                                 "1 KiB",
		1025:                                 "1.001 KiB",
		2048:                                 "2 KiB",
		1024 * 1024:                          "1 MiB",
		1024*1024 - 1:                        "1023.999 KiB",
		2 * 1024 * 1024 * 1024:               "2 GiB",
		3 * 1024 * 1024 * 1024 * 1024:        "3 TiB",
		4 * 1024 * 1024 * 1024 * 1024 * 1024: "4 PiB",
	}
	for n, s := range h {
		if humanize(n) != s {
			t.Errorf("humanize(%d) gives incorrect result: %v instead of %v", n, humanize(n), s)
		}
	}
}
