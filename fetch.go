// Copyright 2016 Cognifly and Contributors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cognibot

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cognifly/cognilog"
	"github.com/kampsy/collectlinks"
)

// The robots.txt relative path
var robotPath, _ = url.Parse("/robots.txt")

const (
	// DefaultCrawlDelay represents the delay to use if there is no robots.txt
	// specified delay.
	DefaultCrawlDelay = 5 * time.Second

	// DefaultUserAgent is the default user agent string.
	DefaultUserAgent = "Cognibot (https://github.com/Cognifly/cognibot)"
)

// Doer defines the method required to use a type as HttpClient.
// The net/*http.Client type satisfies this interface.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// A Fetch defines the parameters for running a web crawler.
type Fetch struct {
	// Default delay to use between requests to a same host if there is no robots.txt
	// crawl delay or if DisablePoliteness is true.
	CrawlDelay time.Duration

	// The *http.Client to use for the requests. If nil, defaults to the net/http
	// package's default client. Should be HTTPClient to comply with go lint, but
	// this is a breaking change, won't fix.
	HTTPClient Doer

	// The user-agent string to use for robots.txt validation and URL fetching.
	UserAgent string

	// keeep track of all the hostnames and their robots.txt
	HostInfo []*Robot

	// number of crawl threads
	crawCount int

	// the frontier or Queue
	mu        sync.RWMutex
	Queue     []Cmder
	index     int // previous url index
	HostCount []string

	// the url visited by crawl threads.
	tex     sync.RWMutex
	Visited []Cmder
}

// New returns an initialized Fetch.
func New() *Fetch {
	// Create a Transport for control over proxies, TLS configuration,
	// keep-alives, compression, and other settings.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	Client := http.DefaultClient
	Client.Transport = tr
	return &Fetch{
		CrawlDelay: DefaultCrawlDelay,
		HTTPClient: Client,
		UserAgent:  DefaultUserAgent,
	}
}

// DoRequest Prepares and executes an http request.
func (f *Fetch) DoRequest(cmd Cmder) (*http.Response, error) {
	req, err := http.NewRequest(cmd.Method(), cmd.URL().String(), nil)
	if err != nil {
		return nil, err
	}
	// If there was no User-Agent implicitly set by the HeaderProvider,
	// set it to the default value.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", f.UserAgent)
	}
	// Do the request.
	res, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// appends the RootURL(seeds) to the Queue or Frontier
func (f *Fetch) addSeed(cmd Cmder) {
	f.mu.Lock()
	f.Queue = append(f.Queue, cmd)
	f.HostCount = append(f.HostCount, cmd.URL().Host)
	f.mu.Unlock()
}

// Seed creates a robot type, appends it to hostinfo and appends RootURL to
// the Queue.
func (f *Fetch) Seed(args ...string) {
	for _, str := range args {
		res, err := f.DoRequest(BotCmd(str))
		if err != nil {
			cognilog.Log("red", err)
		}
		robot := MakeBot(res)
		if !robot.FullDisallow { // if not FullDisallow add
			f.HostInfo = append(f.HostInfo, robot)
			f.addSeed(NewCmd(str)) // add RootURL to Queue or Frontier.
		}
	}
}

// check if a url is present in a slice of cmd's
func checkURL(sl []Cmder, url *url.URL) bool {
	for _, cmd := range sl {
		if cmd.URL().String() == url.String() {
			return true
		}
	}
	return false
}

// edit the url of the fetched page. For UNIX like systems
// sake, remove all / and replace them with _-_
func docName(url *url.URL) string {
	slice := strings.Split(url.String(), "/")
	name := strings.Join(slice, "_-_")
	return name
}

// robExcl finds the url disallowed or allowed by the admin.
// Two key points to take note of.
// One if a robot defines "disallow: /" .Only look out for rules defined with
// the prefix "allow:" .This is so because all disallow: protocals are overwritten
// by "disallow: /".
// Two if a robot defines "allow: /" .Only look out for rules defined with
// the prefix "disallow:" .This is so because all allow: protocals are overwritten
// by "allow: /"
func robExcl(cmd Cmder, info []*Robot) bool {
	var rootExcl bool
	var state bool
	for _, rob := range info {
		if slice, ok := rob.Groups["user-agent:*"]; ok {
			if slice[0] == "disallow:/" {
				rootExcl = false
				state = true
				for num := 1; num < len(slice); num++ {
					trim := strings.TrimPrefix(slice[num], "allow:/")
					exclu, err := parseCmd(trim, rob.RootURL)
					if err != nil {
						return false
					}
					if cmd.URL().String() == exclu.URL().String() {
						return true
					}
				}
			} else if slice[0] == "allow:/" {
				rootExcl = true
				state = true
				for _, pro := range slice {
					trim := strings.TrimPrefix(pro, "disallow:/")
					exclu, err := parseCmd(trim, rob.RootURL)
					if err != nil {
						return false
					}
					if cmd.URL().String() == exclu.URL().String() {
						return false
					}
				}
			}
			// if state is false meaning the rules above have can not be used.
			if !state {
				for _, pro := range slice {
					rootExcl = true
					if strings.HasPrefix(pro, "disallow:/") {
						trim := strings.TrimPrefix(pro, "disallow:/")
						exclu, err := parseCmd(trim, rob.RootURL)
						if err != nil {
							return false
						}
						if cmd.URL().String() == exclu.URL().String() {
							return false
						}
					} else if strings.HasPrefix(pro, "allow:/") {
						trim := strings.TrimPrefix(pro, "disallow:/")
						exclu, err := parseCmd(trim, rob.RootURL)
						if err != nil {
							return false
						}
						if cmd.URL().String() == exclu.URL().String() {
							return true
						}
					}
				}
			}
		}
	}
	return rootExcl
}

func count(sl []string, str string) int {
	var num int
	for _, s := range sl {
		if s == str {
			num++
		}
	}
	return num
}

func filter(cmd Cmder) bool {
	str := strings.ToLower(cmd.URL().String())
	if strings.Contains(str, ".mp3") {
		return true
	} else if strings.HasSuffix(str, ".pdf") {
		return true
	} else if strings.HasSuffix(str, "doc") {
		return true
	} else if strings.Contains(str, "mp4") {
		return true
	}
	return false
}

// crawl thread that gets the body of a web pages and do lots of things with it,
// such as collectiong all the links from it and saving it to disk for future use.
func (f *Fetch) crawl(cr int) {
	for {
		// Get cmd from the queue and update the index num once done.
		var lnk Cmder
		f.mu.RLock()
		if len(f.Queue) == 0 { // if the queue is empty
			cognilog.FatalINFO("red", "panic", "Empty Queue exiting Now!")
		} else if len(f.Queue) == f.index {
			break
		}
		num := f.index
		lnk = f.Queue[num]
		num++
		f.index = num
		f.mu.RUnlock()
		cognilog.LogINFO("cyan", lnk.Method(), fmt.Sprintf("%v", lnk.URL().String()))
		res, err := f.DoRequest(lnk)
		if err != nil || res.StatusCode == 404 {
			if err == nil {
				cognilog.LogINFO("red", fmt.Sprintf("Crawl %d 404", cr), " [Page not found]")
				continue
			}
			cognilog.LogINFO("red", fmt.Sprintf("Crawl %d [Request error]", cr), err)
			continue
		}
		// write the page to disk
		byt, err := ioutil.ReadAll(res.Body)
		if err != nil {
			cognilog.LogINFO("red", fmt.Sprintf("Crawl %d body", cr), err)
			return
		}
		err = ioutil.WriteFile("docs/"+docName(lnk.URL()), byt, 0755)
		if err != nil {
			cognilog.LogINFO("red", fmt.Sprintf("Crawl %d write ", cr), err)
			return
		}
		redr := strings.NewReader(string(byt))
		anchColl := collectlinks.All(redr)
		for _, a := range anchColl {
			cmd, err := parseCmd(a, lnk.URL())
			if err != nil {
				continue
			}

			// lock queue again
			f.mu.RLock()
			if checkURL(f.Queue, cmd.URL()) { // if the url is present in the queue, continue
				cognilog.LogINFO("magenta", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [Already in the Queue skip.]", cmd.URL().String()))
				f.mu.RUnlock() // unlock before continue
				continue
			}

			// robot exclusion
			if !robExcl(cmd, f.HostInfo) {
				cognilog.LogINFO("magenta", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [Disallowed by robot.]", cmd.URL().String()))
				f.mu.RUnlock() // unlock before continue
				continue
			}

			if filter(cmd) {
				cognilog.LogINFO("red", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [Not Accepted]", cmd.URL().String()))
				f.mu.RUnlock() // unlock before continue
				continue
			}

			func(cd Cmder) { // appends cmd to Queue
				var host bool
				for _, rob := range f.HostInfo {
					if rob.RootURL.Host == cd.URL().Host {
						host = true
						break
					}
				}
				if host == false {
					cognilog.LogINFO("magenta", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [Not found in HostInfo Hash table]", cd.URL()))
					f.mu.RUnlock()
					return
				}

				if count(f.HostCount, cd.URL().Host) > 20 {
					cognilog.LogINFO("cyan", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [maxed out no slot available]", cd.URL()))
					f.mu.RUnlock()
					return
				}
				f.HostCount = append(f.HostCount, cd.URL().Host)

				f.Queue = append(f.Queue, cd)
				cognilog.LogINFO("green", fmt.Sprintf("Crawl %d", cr), fmt.Sprintf("%v [Appended to Queue]", cd.URL().String()))
				f.mu.RUnlock()
			}(cmd)
		}

		// append cmd to Visited
		f.tex.RLock()
		f.Visited = append(f.Visited, lnk)
		f.tex.RUnlock()

		time.Sleep(f.CrawlDelay)
	}
}

// watch over f.Queue and f.Visited.
func (f *Fetch) watch(c chan bool) {
	var deadLock int
	var str string
	for {
		queue := len(f.Queue)
		visit := len(f.Visited)
		if visit > 0 && queue > 0 {
			if visit >= queue {
				c <- true
				break
			}
		}
		if deadLock > 5 {
			c <- true
			break
		}
		if fmt.Sprintf("%d%d", queue, visit) == str {
			deadLock++
		}
		str = fmt.Sprintf("%d%d", queue, visit)

		if queue-visit > 10 {
			num := f.crawCount + 1
			go f.crawl(num)
			f.crawCount = num
			cognilog.LogINFO("cyan", "go crwal", fmt.Sprintf("Started crawl thread [%d]", num))
		}

		cognilog.LogINFO("yellow", "status", fmt.Sprintf("Queue[%d]  Visited[%d]", queue, visit))
		time.Sleep(time.Duration(1) * time.Second)
	}
}

// Start runs all the specified crawl goroutines.
func (f *Fetch) Start(num int) {
	c := make(chan bool)
	go f.watch(c)
	for i := 1; i <= num; i++ {
		go f.crawl(i)
		f.crawCount = f.crawCount + i
	}
	for {
		if <-c {
			cognilog.LogINFO("yellow", f.UserAgent, "Closed all Crawl threads.")
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}
