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
	"os/exec"
	"strconv"
	"strings"
)

// Return the maximum number of supported NLinks to pathname.
// Since the syscall interface to Pathconf isn't supported on all unixes (such
// as Linux, for some reason), we instead call out to the getconf program,
// which should always be available as a basic command on both BSDs and Linux,
// to obtain the value.  Since this only needs to be done once per device (ie.
// once per Stat_t.Dev), it isn't a performance concern.
func MaxNlink(pathname string) uint64 {
	var returnVal uint64
	var cmdPath string
	var err error

	returnVal = 8 // Minimum supported MAX_LINK
	if cmdPath, err = exec.LookPath("/bin/getconf"); err == nil {
		cmdPath = "/bin/getconf"
	} else if cmdPath, err = exec.LookPath("/usr/bin/getconf"); err == nil {
		cmdPath = "/usr/bin/getconf"
	} else {
		// Try Pathconf()? on darwin/BSD before giving up?
		return returnVal
	}

	cmd := exec.Command(cmdPath, "LINK_MAX", pathname)
	out, err := cmd.Output()
	if err != nil {
		return returnVal
	}

	outStr := strings.TrimSpace(string(out))

	maxNlinks, err := strconv.ParseUint(outStr, 10, 64)
	if err != nil {
		return returnVal
	}

	return maxNlinks
}
