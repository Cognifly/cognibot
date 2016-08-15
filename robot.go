//Package cognibot Copyright 2016 Cognifly and Contributors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cognibot

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cognifly/cognilog"
)

// Robot passed exclusion for host
type Robot struct {
	RootURL      *url.URL
	FullAllow    bool
	FullDisallow bool
	Groups       map[string][]string
	CrawDelay    time.Duration
}

// Removes the whitespaces.
func trimSpaces(s string) string {
	reg := regexp.MustCompile("\\s")
	clear := reg.ReplaceAllString(s, " ")
	rmSpaces := strings.Split(clear, " ")
	var str string
	for _, s := range rmSpaces {
		str = str + s
	}
	return str
}

// are all user-agents allowed?
func isAllowAll(grp map[string][]string) bool {
	if len(grp) == 1 {
		for _, val := range grp {
			if len(val) == 1 {
				if trimSpaces(val[0]) == "allow:/" {
					return true
				}
			}
		}
	}
	return false
}

// are all user-agents disallowed?
func isDisallowAll(grp map[string][]string) bool {
	if len(grp) == 1 {
		for _, val := range grp {
			if len(val) == 1 {
				if trimSpaces(val[0]) == "disallow:/" {
					return true
				}
			}
		}
	}
	return false
}

// MakeBot takes an http.Response and returns a Robot
func MakeBot(res *http.Response) *Robot {
	var robot = new(Robot)
	// treat all 4xx errors in the same way. Assume that there are no restrictions.
	if res.StatusCode >= 400 && res.StatusCode > 500 {
		robot.RootURL = NewCmd(strings.TrimSuffix(res.Request.URL.String(), "robots.txt")).URL()
		robot.FullAllow = true
		robot.CrawDelay = DefaultCrawlDelay
		return robot
	} else if res.StatusCode == 200 {
		byt, err := ioutil.ReadAll(res.Body)
		if err != nil {
			cognilog.LogINFO("red", "Body read error", err)
		}
		redr := strings.NewReader(string(byt))
		scanner := bufio.NewScanner(redr)
		var groups = make(map[string][]string)
		var key string
		var status bool
		for scanner.Scan() {
			txt := strings.ToLower(scanner.Text())
			if txt == "" {
				continue
			}
			// new group
			if strings.HasPrefix(txt, "user-agent:") {
				key = trimSpaces(txt)
				status = true
				continue
			}
			if status && key != "" {
				groups[key] = append(groups[key], trimSpaces(txt))
			}
		}
		if err := scanner.Err(); err != nil {
			cognilog.Log("red", err)
		}
		if isAllowAll(groups) {
			robot.FullAllow = true
		} else if isDisallowAll(groups) {
			robot.FullDisallow = true
		}
		robot.RootURL = NewCmd(strings.TrimSuffix(res.Request.URL.String(), "robots.txt")).URL()
		robot.Groups = groups
		robot.CrawDelay = DefaultCrawlDelay
		return robot
	}

	robot.RootURL = NewCmd(strings.TrimSuffix(res.Request.URL.String(), "robots.txt")).URL()
	return robot
}
