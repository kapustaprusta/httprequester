package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	requestTimeout = 3
)

var (
	toolName         = ""
	httpPrefix       = "http://"
	httpsPrefix      = "https://"
	messageAboutHelp = "Use \"%s --help\" for more information about a tool."
)

func doWork(client *http.Client, urlsChan <-chan string, resultsChan chan<- string, errsChan chan<- error) {
	for url := range urlsChan {
		response, err := client.Get(url)
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
}

func run(parallelWorkers int, urlsToVisit []string) (<-chan string, <-chan error) {
	errsCh := make(chan error, parallelWorkers)
	resultsCh := make(chan string, parallelWorkers)
	urlsToVisitCh := make(chan string, parallelWorkers)

	for workerIdx := 0; workerIdx < parallelWorkers; workerIdx++ {
		client := &http.Client{
			Timeout: time.Second * requestTimeout,
		}
		go doWork(client, urlsToVisitCh, resultsCh, errsCh)
	}

	go func() {
		for _, urlToVisit := range urlsToVisit {
			urlsToVisitCh <- urlToVisit
		}
		close(urlsToVisitCh)
	}()

	return resultsCh, errsCh
}

func repairUrls(urls []string) []string {
	var repairedUrls []string
	for _, repairedUrl := range urls {
		if !strings.HasPrefix(repairedUrl, httpPrefix) && !strings.HasPrefix(repairedUrl, httpsPrefix) {
			repairedUrls = append(repairedUrls, httpPrefix+repairedUrl)
		}
	}

	return repairedUrls
}

func validateUrls(urls []string) ([]string, []string) {
	var validUrls []string
	var invalidUrls []string

	for _, validatedUrl := range urls {
		_, err := url.ParseRequestURI(validatedUrl)
		if err != nil {
			invalidUrls = append(invalidUrls, validatedUrl)
		} else {
			validUrls = append(validUrls, validatedUrl)
		}
	}

	return validUrls, invalidUrls
}

func validateParams(parallelWorkers int, urlsToVisit []string) error {
	if parallelWorkers < 1 {
		return fmt.Errorf("Invalid number of parallel workers. " + fmt.Sprintf(messageAboutHelp, toolName))
	}

	if len(urlsToVisit) == 0 {
		return fmt.Errorf("List of urls to visit is empty. " + fmt.Sprintf(messageAboutHelp, toolName))
	}

	return nil
}

func parseParams() (int, []string) {
	toolName = os.Args[0]
	parallelWorkers := flag.Int("parallel", 10, "number of parallel workers")

	flag.Parse()
	urlsToVisit := flag.Args()

	return *parallelWorkers, urlsToVisit
}

func init() {
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("  url1 url2 ...\n\tlist of urls to visit separated by whitespace (default empty)")
	}
}

func main() {
	parallelWorkers, urlsToVisit := parseParams()
	err := validateParams(parallelWorkers, urlsToVisit)
	if err != nil {
		fmt.Println(err)

		return
	}

	validUrls, invalidUrls := validateUrls(urlsToVisit)
	if len(invalidUrls) != 0 {
		validUrls, invalidUrls = validateUrls(repairUrls(urlsToVisit))
		for _, invalidUrl := range invalidUrls {
			fmt.Printf("Invalid url: \"%s\"\n", invalidUrl)
		}
	}

	resultsCh, errsCh := run(parallelWorkers, validUrls)
	for validUrlIdx := 0; validUrlIdx < len(validUrls); validUrlIdx++ {
		select {
		case res := <-resultsCh:
			fmt.Println(res)
		case err := <-errsCh:
			fmt.Println(err)
		}
	}
}
