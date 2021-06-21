package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	urls     []string
	parallel *int

	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

const (
	requestTimeOut = 3
)

func doWork(urlsChan <-chan string) (<-chan string, <-chan error) {
	errsChan := make(chan error, 1)
	resultsChan := make(chan string, 1)
	httpClient := http.Client{
		Timeout: time.Second * requestTimeOut,
	}

	go func() {
		for url := range urlsChan {
			response, err := httpClient.Get(url)
			if err != nil {
				errsChan <- err

				continue
			}

			rawResponseBody, err := ioutil.ReadAll(response.Body)
			if err != nil {
				errsChan <- err

				continue
			}
			response.Body.Close()

			hashSum := md5.Sum(rawResponseBody)
			resultsChan <- fmt.Sprintf("%s %x", url, hashSum)
		}

		close(errsChan)
		close(resultsChan)
	}()

	return resultsChan, errsChan
}

func checkPrefix(urls []string) {
	for urlIdx, url := range urls {
		if !strings.HasPrefix(url, httpPrefix) || !strings.HasPrefix(url, httpsPrefix) {
			urls[urlIdx] = httpPrefix + url
		}
	}
}

func init() {
	parallel = flag.Int("parallel", 10, "number of parallel workers")
}

func main() {
	flag.Parse()
	urls = flag.Args()

	urlsChan := make(chan string, len(urls))
	errsChans := make([]<-chan error, *parallel)
	resultsChans := make([]<-chan string, *parallel)

	for workerIdx := 0; workerIdx < *parallel; workerIdx++ {
		resultsChan, errsChan := doWork(urlsChan)
		errsChans[workerIdx] = errsChan
		resultsChans[workerIdx] = resultsChan
	}

	checkPrefix(urls)
	for _, url := range urls {
		urlsChan <- url
	}
	close(urlsChan)

	for _, errsChan := range errsChans {
		for err := range errsChan {
			fmt.Println(err)
		}
	}

	for _, resultsChan := range resultsChans {
		for result := range resultsChan {
			fmt.Println(result)
		}
	}
}
