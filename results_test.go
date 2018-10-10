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
		if Humanize(n) != s {
			t.Errorf("Humanize(%d) gives incorrect result: %v instead of %v", n, Humanize(n), s)
		}
	}
}

func TestHumanizedUint64(t *testing.T) {
	h := map[string]uint64{
		"0":    0,
		"0k":   0,
		"0K":   0,
		"0p":   0,
		"1":    1,
		"1023": 1023,
		"1k":   1024,
		"1025": 1025,
		"2k":   2048,
		"1m":   1024 * 1024,
		"2M":   2 * 1024 * 1024,
		"2g":   2 * 1024 * 1024 * 1024,
		"3t":   3 * 1024 * 1024 * 1024 * 1024,
		"4p":   4 * 1024 * 1024 * 1024 * 1024 * 1024,
	}
	for s, n := range h {
		v, err := HumanizedUint64(s)
		if err != nil {
			t.Errorf("HumanizedUint64(%s) gives error result: %v", s, err)
		}
		if v != n {
			t.Errorf("HumanizedUint64(%s) gives incorrect result: %v instead of %v",
				s, v, n)
		}
	}
	h = map[string]uint64{
		"k":  0,
		"K":  0,
		"kk": 0,
		"aK": 0,
		"bp": 0,
	}
	for s, _ := range h {
		v, err := HumanizedUint64(s)
		if err == nil {
			t.Errorf("HumanizedUint64(%s) should give error result, got: %v", s, v)
		}
	}
}
