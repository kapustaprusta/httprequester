package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"
)

func DummyServer(w http.ResponseWriter, r *http.Request) {
	switch r.RequestURI {
	case "/testrequest1":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sed ut perspiciatis unde omnis iste natus error"))
	case "/testrequest2":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("quae ab illo inventore veritatis et quasi architecto"))
	case "/testrequest3":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("aspernatur aut odit aut fugit, sed quia consequuntur"))
	}
}

func DummyServerWithLongTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * requestTimeout * 2)
	DummyServer(w, r)
}

func NewDummyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(DummyServer))
}

func NewDummyServerWithLongTimeout() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(DummyServerWithLongTimeout))
}

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name            string
		parallelWorkers int
		urlsToVisit     []string
		err             error
	}{
		{
			name:            "valid",
			parallelWorkers: 10,
			urlsToVisit: []string{
				"https://www.test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			err: nil,
		},
		{
			name:            "invalid number of parallel workers",
			parallelWorkers: 0,
			urlsToVisit: []string{
				"https://www.test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			err: fmt.Errorf("Invalid number of parallel workers. " + fmt.Sprintf(messageAboutHelp, toolName)),
		},
		{
			name:            "empty list of urls to visit",
			parallelWorkers: 10,
			urlsToVisit:     []string{},
			err:             fmt.Errorf("List of urls to visit is empty. " + fmt.Sprintf(messageAboutHelp, toolName)),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualErr := validateParams(testCase.parallelWorkers, testCase.urlsToVisit)
			if testCase.err == nil {
				if testCase.err != actualErr {
					t.Fatalf("Wrong error value. Expected: %v. Actual: %s.", testCase.err, actualErr)
				}
			} else {
				if testCase.err.Error() != actualErr.Error() {
					t.Fatalf("Wrong error value. Expected: %s. Actual: %s.", testCase.err, actualErr)
				}
			}
		})
	}
}

func TestValidateUrls(t *testing.T) {
	testCases := []struct {
		name        string
		urls        []string
		validUrls   []string
		invalidUrls []string
	}{
		{
			name: "valid",
			urls: []string{
				"https://www.test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			validUrls: []string{
				"https://www.test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			invalidUrls: []string{},
		},
		{
			name: "url without https prefix",
			urls: []string{
				"www.test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			validUrls: []string{
				"https://www.test2.com",
				"https://www.test3.com",
			},
			invalidUrls: []string{
				"www.test1.com",
			},
		},
		{
			name: "url without prefixes",
			urls: []string{
				"test1.com",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			validUrls: []string{
				"https://www.test2.com",
				"https://www.test3.com",
			},
			invalidUrls: []string{
				"test1.com",
			},
		},
		{
			name: "url without domain",
			urls: []string{
				"https://www.test1",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			validUrls: []string{
				"https://www.test1",
				"https://www.test2.com",
				"https://www.test3.com",
			},
			invalidUrls: []string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualValidUrls, actualInvalidUrls := validateUrls(testCase.urls)
			if len(testCase.validUrls) != 0 || len(actualValidUrls) != 0 {
				if !reflect.DeepEqual(testCase.validUrls, actualValidUrls) {
					t.Fatalf("Wrong list of valid urls. Expected: %v. Actual: %v.", testCase.validUrls, actualValidUrls)
				}
			}
			if len(testCase.invalidUrls) != 0 || len(actualInvalidUrls) != 0 {
				if !reflect.DeepEqual(testCase.invalidUrls, actualInvalidUrls) {
					t.Fatalf("Wrong list of invalid urls. Expected: %v. Actual: %v.", testCase.invalidUrls, actualInvalidUrls)
				}
			}
		})
	}
}

func TestDoWork(t *testing.T) {
	server := NewDummyServer()
	defer server.Close()

	serverWithTimeout := NewDummyServerWithLongTimeout()
	defer serverWithTimeout.Close()

	urls := []string{
		server.URL + "/testrequest1",
		server.URL + "/testrequest2",
		server.URL + "/testrequest3",
	}

	parallelWorkers := 3

	var results []string
	for _, url := range urls {
		resp, _ := http.Get(url)
		respBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		results = append(results, fmt.Sprintf("%s %x", url, md5.Sum(respBody)))
	}

	testCases := []struct {
		name    string
		client  *http.Client
		urls    []string
		results []string
		errs    []string
	}{
		{
			name:    "valid",
			client:  http.DefaultClient,
			urls:    urls,
			results: results,
			errs:    []string{},
		},
		{
			name:    "invalid url",
			client:  http.DefaultClient,
			urls:    append([]string{"testrequest1"}, urls[1:]...),
			results: results[1:],
			errs: []string{
				"Get \"testrequest1\": unsupported protocol scheme \"\"",
			},
		},
		{
			name: "long request",
			client: &http.Client{
				Timeout: time.Second * requestTimeout,
			},
			urls:    append([]string{serverWithTimeout.URL + "/testrequest1"}, urls[1:]...),
			results: results[1:],
			errs: []string{
				fmt.Sprintf("Get \"%s/testrequest1\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)", serverWithTimeout.URL),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			errsCh := make(chan error, parallelWorkers)
			resultsCh := make(chan string, parallelWorkers)
			urlsCh := make(chan string, parallelWorkers)

			for workerIdx := 0; workerIdx < parallelWorkers; workerIdx++ {
				go doWork(testCase.client, urlsCh, resultsCh, errsCh)
			}

			go func() {
				for _, url := range testCase.urls {
					urlsCh <- url
				}
				close(urlsCh)
			}()

			var actualResults []string
			var actualErrs []string

			for urlIdx := 0; urlIdx < len(testCase.urls); urlIdx++ {
				select {
				case res := <-resultsCh:
					actualResults = append(actualResults, res)
				case err := <-errsCh:
					actualErrs = append(actualErrs, err.Error())
				}
			}

			sort.Strings(testCase.results)
			sort.Strings(testCase.errs)

			sort.Strings(actualResults)
			sort.Strings(actualErrs)

			if len(testCase.results) != 0 || len(actualResults) != 0 {
				if !reflect.DeepEqual(testCase.results, actualResults) {
					t.Fatalf("Wrong results. Expected: %v. Actual: %v.", testCase.results, actualResults)
				}
			}
			if len(testCase.errs) != 0 || len(actualErrs) != 0 {
				if !reflect.DeepEqual(testCase.errs, actualErrs) {
					t.Fatalf("Wrong errors. Expected: %v. Actual: %v.", testCase.errs, actualErrs)
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	server := NewDummyServer()
	defer server.Close()

	serverWithTimeout := NewDummyServerWithLongTimeout()
	defer serverWithTimeout.Close()

	urls := []string{
		server.URL + "/testrequest1",
		server.URL + "/testrequest2",
		server.URL + "/testrequest3",
	}

	var results []string
	for _, url := range urls {
		resp, _ := http.Get(url)
		respBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		results = append(results, fmt.Sprintf("%s %x", url, md5.Sum(respBody)))
	}

	testCases := []struct {
		name            string
		parallelWorkers int
		urls            []string
		results         []string
		errs            []string
	}{
		{
			name:            "valid",
			parallelWorkers: 3,
			urls:            urls,
			results:         results,
			errs:            []string{},
		},
		{
			name:            "invalid url",
			parallelWorkers: 3,
			urls:            append([]string{"testrequest1"}, urls[1:]...),
			results:         results[1:],
			errs: []string{
				"Get \"testrequest1\": unsupported protocol scheme \"\"",
			},
		},
		{
			name:            "long request",
			parallelWorkers: 3,
			urls:            append([]string{serverWithTimeout.URL + "/testrequest1"}, urls[1:]...),
			results:         results[1:],
			errs: []string{
				fmt.Sprintf("Get \"%s/testrequest1\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)", serverWithTimeout.URL),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resultsCh, errsCh := run(testCase.parallelWorkers, testCase.urls)

			var actualResults []string
			var actualErrs []string

			for urlIdx := 0; urlIdx < len(testCase.urls); urlIdx++ {
				select {
				case res := <-resultsCh:
					actualResults = append(actualResults, res)
				case err := <-errsCh:
					actualErrs = append(actualErrs, err.Error())
				}
			}

			sort.Strings(testCase.results)
			sort.Strings(testCase.errs)

			sort.Strings(actualResults)
			sort.Strings(actualErrs)

			if len(testCase.results) != 0 || len(actualResults) != 0 {
				if !reflect.DeepEqual(testCase.results, actualResults) {
					t.Fatalf("Wrong results. Expected: %v. Actual: %v.", testCase.results, actualResults)
				}
			}
			if len(testCase.errs) != 0 || len(actualErrs) != 0 {
				if !reflect.DeepEqual(testCase.errs, actualErrs) {
					t.Fatalf("Wrong errors. Expected: %v. Actual: %v.", testCase.errs, actualErrs)
				}
			}
		})
	}
}
