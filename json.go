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
	"encoding/json"
	"fmt"
	P "hardlinkable/internal/pathpool"
	"time"
)

type jsonStats struct {
	ExistingLinks     map[string][]string `json:"existingLinks"`
	ExistingLinkSizes map[string]uint64   `json:"existingLinkSizes"`
	LinkPaths         [][]string          `json:"linkPaths"`
	CountingStats
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	RunTime   string    `json:"runTime"`
}

func (ls *linkingStats) OutputJSONResults() {
	duration := ls.EndTime.Sub(ls.StartTime)
	jstats := jsonStats{
		CountingStats: ls.CountingStats,
		StartTime:     ls.StartTime,
		EndTime:       ls.EndTime,
		RunTime:       duration.Round(time.Millisecond).String(),
	}

	existingLinks := make(map[string][]string)
	for src, v := range ls.ExistingLinks {
		dsts := make([]string, 0, len(v.paths))
		for _, pathsplit := range v.paths {
			dsts = append(dsts, pathsplit.Join())
		}
		existingLinks[src.Join()] = dsts
	}
	jstats.ExistingLinks = existingLinks

	existingLinkSizes := make(map[string]uint64)
	for src, v := range ls.ExistingLinks {
		existingLinkSizes[src.Join()] = v.size
	}
	jstats.ExistingLinkSizes = existingLinkSizes

	var links []string
	linkPaths := make([][]string, 0)
	prevPathsplit := P.Pathsplit{}
	for _, p := range ls.LinkPairs {
		if p.Src != prevPathsplit {
			if len(links) > 0 {
				linkPaths = append(linkPaths, links)
			}
			links = []string{p.Src.Join()}
			prevPathsplit = p.Src
		}
		links = append(links, p.Dst.Join())
	}
	if len(links) > 0 {
		linkPaths = append(linkPaths, links)
	}
	jstats.LinkPaths = linkPaths

	b, _ := json.Marshal(jstats)
	fmt.Println(string(b))
}
