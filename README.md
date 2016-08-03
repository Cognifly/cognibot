<div style="width: 100%;" align="center">
  <div style="font: 40pt Roboto, sans-serif;font-weight: 300; color: #CC181E;">
    Cognibot
  </div>
  <div style="margin-top: 15px;font: 15px Roboto, sans-serif;font-weight: 300; color: #21252B;">
    Beta version
  </div>

  <div style="max-width: 500px;margin-top: 15px;font: 15px Roboto, sans-serif;font-weight: 300; color: #21252B;">
    Thank you for checking out cognibot. To help us improve, fell free to fork the repository and send us a pull request.
  </div>
  <div style="margin-top: 15px;margin-bottom: 15px;font: 15px Roboto, sans-serif;font-weight: 300; color: #21252B;">
    Crafted with love By Kampamba Chanda.
  </div>
 </div>
 
# cognibot [![build status](https://secure.travis-ci.org/cognifly/cognibot.png)](http://travis-ci.org/cognifly/cognibot) [![GoDoc](https://godoc.org/github.com/cognifly/cognibot)](http://godoc.org/github.com/cognifly/cognibot)

Package cognibot provides a simple and concurrent web crawler that follows the robots.txt
policies and crawl delays.

## Note
By default, cognibot only crwals URLs with the same host names as the seeds.
Cognibot is very much inspired by of [fetchbot](https://github.com/PuerkitoBio/fetchbot).


## Installation

To install, simply run in a terminal:

    go get github.com/cognifly/cognibot

The package has two dependencies, [cognilog](https://github.com/cognifly/cognilog) a simple but 
effective color logging package and [collectlinks](https://github.com/kampsy/collectlinks) a 
package that collects all URLs from a webpage. 

The [API documentation is available on godoc.org](http://godoc.org/github.com/cognifly/cognibot).

## Usage

The following example (taken from /example/main.go) shows how to create a Fetch, add seeds and
start the crwal. 

```go
package main

import (
	"github.com/cognifly/cognibot"
)

func main() {
	fetch := cognibot.New()
	fetch.Seed("http://localhost:2016/")

	fetch.Start()
}
```


### Cognibot

Basically, **Cognibot** has the following functionality:

A **Fetch** module that uses the http protocal to retrieve the robot.txt or 
web page at a URL.

A built in **Robot** exclusion protocal parser that determines whether an extracted 
link passes the robot restrictions.

A **Queue or URL Frontier** containing URLs yet to be fetched by the crwal thread or 
threads.

A **Visited** list that logs all the URLs fetched by the fecth mobule.

A **Parser** module that extracts links from a fetched webpage and saves the webpage to 
disk.

A **Duplication** elimination module that determines whether an extracted link is already 
in the queue or url frontier 


### How Cognibot Works

Crawling is performed by the a crawl thread or crawl goroutine. Add the number to Start 
specifying the number of threads to begin with. If the number of URL in the queue minus 
the total number in visited is less than 100, cognibot spins up a new thread. see example below.

```go
package main

import (
	"github.com/cognifly/cognibot"
)

func main() {
	fetch := cognibot.New()
	fetch.Seed("http://localhost:2016/", "http://localhost:2017/")

	fetch.Start(3)
}
```
A crawl thread or crawl goroutine begins by taking a URL from the queue or frontier and 
fetching the web page at the URL. The URL is then added to the Visited list.

Links are extacted from the fetched page's response body and then the webpage body 
is saved to disk.

Each extracted link goes through a series of tests to determine whether it should 
be added to the queue or URL frontier. First the crawl thread or crawl goroutine tests 
whether the Host of the URL is Found in the **HostInfo []*Robot**. Secondly it tests 
whether the URL is already in the the queue or URL frontier and Finaly it tests whether the URL 
under consideration passes the robot restrictions.


### Cognibot Options

Cognibot has a number of fields that provide further customization:

* HttpClient : By default, the Fetcher uses the net/http default Client to make requests. A
different client can be set on the Fetcher.HttpClient field.

* CrawlDelay : That value is used only if there is no delay specified
by the robots.txt of a given host.

* UserAgent : Sets the user agent string to use for the requests and to validate
against the robots.txt entries.


## License

The [BSD-style license] found in the LICENSE file, the same as cognilog package. 
The collectlinks package source code is under the [MIT license] (details in
the source file).

[Name]=> kampsy kampamba chanda
[Email]=> kampsycode@gmail.com
[Github]=> https://github.com/kampsy
[Social]=> google.com/+kampambachanda
