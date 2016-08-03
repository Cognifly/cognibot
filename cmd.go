//Package cognibot Copyright 2016 Cognifly and Contributors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cognibot

import (
	"net/url"

	"github.com/cognifly/cognilog"
)

// Cmder interface defines the methods required by the Fetch to request
// a resource.
type Cmder interface {
	URL() *url.URL
	Method() string
}

// Cmd defines a basic Command implementation.
type Cmd struct {
	U *url.URL
	M string
}

// URL returns the resource targeted by this command.
func (c *Cmd) URL() *url.URL {
	return c.U
}

// Method returns the HTTP verb to use to process this command (i.e. "GET", "HEAD", etc.).
func (c *Cmd) Method() string {
	return c.M
}

// NewCmd returns an initialized Cmd
func NewCmd(s string) *Cmd {
	url, err := url.Parse(s)
	if err != nil {
		cognilog.Log("red", err)
	}
	return &Cmd{
		U: url,
		M: "GET",
	}
}

// parseCmd returns cmd by resolving a url
func parseCmd(s string, lnk *url.URL) (*Cmd, error) {
	href, err := url.Parse(s)
	if err != nil {
		cognilog.Log("red", err)
		return nil, err
	}

	url := lnk.ResolveReference(href)
	return &Cmd{
		U: url,
		M: "GET",
	}, nil
}

// BotCmd returns an initialized robot Cmd
func BotCmd(s string) *Cmd {
	lnk, err := url.Parse(s)
	if err != nil {
		cognilog.Log("red", err)
	}
	rob := lnk.ResolveReference(robotPath)
	return &Cmd{
		U: rob,
		M: "GET",
	}
}
