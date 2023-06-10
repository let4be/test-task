package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var timeout time.Duration

type taskStatus struct {
	url  string
	stat stat
	err  error
}

type stat struct {
	ResponseCode *int
	Duration     time.Duration
}

type statusTracker struct {
	total     int
	ok        int
	errors    int
	cancelled int
	codes     map[int]int
}

func newStatusTracker() *statusTracker {
	return &statusTracker{
		codes: make(map[int]int),
	}
}

func (t *statusTracker) printSummary() {
	log.WithFields(log.Fields{
		"TOTAL":     t.total,
		"OK":        t.ok,
		"ERRORS":    t.errors,
		"CANCELLED": t.cancelled,
		"CODES":     t.codes,
	}).Info("Crawling finished")
}

func (t *statusTracker) track(status taskStatus) {
	logger := log.WithField("url", status.url)

	t.total++
	if status.stat.ResponseCode != nil {
		t.codes[*status.stat.ResponseCode]++
	}

	if status.err != nil {
		if status.err == context.Canceled {
			t.cancelled++
			logger.Info("CANCELLED")
		} else {
			t.errors++
			logger.WithError(status.err).Info("NOT OK")
		}
	} else {
		t.ok++
		logger.Info("OK")
	}
}

func newStatWithResponseCode(code int, startedAt time.Time) stat {
	return stat{
		ResponseCode: &code,
		Duration:     time.Since(startedAt),
	}
}

func newStat(startedAt time.Time) stat {
	return stat{
		ResponseCode: nil,
		Duration:     time.Since(startedAt),
	}
}

func head(ctx context.Context, url string) (stat, error) {
	started := time.Now()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return newStat(started), fmt.Errorf("cannot create HEAD request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Unwrap(err) == context.Canceled {
			return newStat(started), context.Canceled
		}
		return newStat(started), fmt.Errorf("cannot HEAD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return newStatWithResponseCode(resp.StatusCode, started), nil
	}

	return newStatWithResponseCode(resp.StatusCode, started), fmt.Errorf("non 200 status code returned: %v", resp.StatusCode)
}

func headWithStatus(ctx context.Context, url string) taskStatus {
	stat, err := head(ctx, url)
	return taskStatus{
		url:  url,
		stat: stat,
		err:  err,
	}
}

func serialExec(ctx context.Context, urls []string) {
	tracker := newStatusTracker()

	for _, url := range urls {
		status := headWithStatus(ctx, url)
		tracker.track(status)
	}

	tracker.printSummary()
}

func parallelExec(ctx context.Context, urls chan string, concurrency int, maxOk int) {
	resCh := make(chan taskStatus)

	cancellableCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		g, gCtx := errgroup.WithContext(cancellableCtx)
		g.SetLimit(concurrency)

		for url := range urls {
			func(url string) {
				g.Go(func() error {
					resCh <- headWithStatus(gCtx, url)
					return nil
				})
			}(url)
		}
		if err := g.Wait(); err != nil {
			log.WithError(err).Warn("Group failed with an error")
		}

		close(resCh)
	}()

	statTracker := newStatusTracker()
	for taskStatus := range resCh {
		statTracker.track(taskStatus)

		if maxOk > 0 && statTracker.ok >= maxOk {
			cancel()
		}
	}

	statTracker.printSummary()
}

func main() {
	var aUrls = flag.String("urls", "", "a file containing a list of urls, one per line")
	var max = flag.Int("max", 2, "max OK fetches")
	var concurrency = flag.Int("concurrency", 3, "max concurrency")
	var timeoutSec = flag.Int("timeout", 5, "timeout in seconds")
	timeout = time.Second * time.Duration(*timeoutSec)
	flag.Parse()

	content, err := os.ReadFile(*aUrls)
	if err != nil {
		log.WithError(err).Fatalf("cannot read urls")
	}
	urls := strings.Split(string(content), "\n")

	ctx := context.Background()

	if *concurrency == 1 {
		serialExec(ctx, urls)
	} else {
		// an idiomatic way to handle option 4 is a channel
		ch := make(chan string)
		go func() {
			// which we close signifying the end of input
			defer close(ch)
			for _, url := range urls {
				ch <- url
			}
		}()
		parallelExec(ctx, ch, *concurrency, *max)
	}
}
