package scraper

import (
	"context"
	"slices"
	"sync"
	"time"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/mo"
	"go.uber.org/atomic"

	"github.com/KonishchevDmitry/feedsd/internal/util"
	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

const scrapePeriod = time.Hour

type Scraper struct {
	feed      feed.Feed
	stat      scrapingStat
	started   chan struct{}
	stopped   chan struct{}
	waitGroup sync.WaitGroup

	lock    util.GuardedLock
	result  mo.Option[scrapeResult]
	waiters []chan<- scrapeResult
}

func newScraper(feed feed.Feed) *Scraper {
	var stat scrapingStat
	stat.feedTime.Store(time.Now())

	return &Scraper{
		feed:    feed,
		stat:    stat,
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}),
	}
}

func (c *Scraper) start(ctx context.Context, develMode bool) {
	if !develMode {
		select {
		case c.started <- struct{}{}:
		default:
		}
	}
	c.waitGroup.Go(func() {
		c.daemon(ctx)
	})
}

func (c *Scraper) stop(ctx context.Context) {
	logging.L(ctx).Infof("Stopping %q scrapper...", c.feed.Name())
	close(c.stopped)
	c.waitGroup.Wait()
	logging.L(ctx).Infof("%q scrapper has stopped.", c.feed.Name())
}

func (c *Scraper) Get(ctx context.Context) (*rss.Feed, error) {
	lock := c.lock.Lock()
	defer lock.UnlockIfLocked()

	if result, ok := c.result.Get(); ok {
		return result.feed, result.error
	}

	waiter := make(chan scrapeResult, 1)
	c.waiters = append(c.waiters, waiter)
	lock.Unlock()

	select {
	case c.started <- struct{}{}:
	default:
	}

	select {
	case result := <-waiter:
		return result.feed, result.error

	case <-c.stopped:
		return nil, context.Canceled

	case <-ctx.Done():
		lock.Lock()
		if index := slices.Index(c.waiters, waiter); index != -1 {
			c.waiters = slices.Delete(c.waiters, index, index+1)
		}
		return nil, ctx.Err()
	}
}

func (c *Scraper) Collect(metrics chan<- prometheus.Metric) {
	metrics <- prometheus.MustNewConstMetric(
		feedAgeMetric, prometheus.GaugeValue,
		time.Since(c.stat.feedTime.Load()).Seconds(), c.feed.Name())

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.success.Load()), c.feed.Name(), "success")

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.unavailable.Load()), c.feed.Name(), "unavailable")

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.error.Load()), c.feed.Name(), "error")
}

func (c *Scraper) daemon(ctx context.Context) {
	select {
	case <-c.started:
	case <-c.stopped:
		return
	}

	// FIXME(konishchev): Random delay?
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
		case <-c.stopped:
			return
		}

		logging.L(ctx).Infof("Scraping %q feed...", c.feed.Name())
		feed, err := c.feed.Get(ctx)
		result := scrapeResult{
			feed:  feed,
			error: err,
		}
		if err == nil {
			logging.L(ctx).Infof("%q feed scraped.", c.feed.Name())
			c.stat.feedTime.Store(time.Now())
			c.stat.success.Inc()
		} else if util.IsTemporaryError(err) {
			logging.L(ctx).Warnf("Failed to scrape %q feed: %s.", c.feed.Name(), err)
			c.stat.unavailable.Inc()
		} else {
			logging.L(ctx).Errorf("Failed to scrape %q feed: %s.", c.feed.Name(), err)
			c.stat.error.Inc()
		}

		lock := c.lock.Lock()
		c.result = mo.Some(result)
		waiters := c.waiters
		c.waiters = nil
		lock.Unlock()

		for _, waiter := range waiters {
			waiter <- result
		}

		timer.Reset(scrapePeriod)
	}
}

type scrapeResult struct {
	feed  *rss.Feed
	error error
}

type scrapingStat struct {
	feedTime    atomic.Time
	success     atomic.Int64
	unavailable atomic.Int64
	error       atomic.Int64
}
