package scraper

import (
	"context"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/samber/mo"

	"github.com/KonishchevDmitry/feedsd/internal/util"
	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

const scrapePeriod = time.Hour

type Scraper struct {
	feed    feed.Feed
	metrics observers

	force     chan struct{}
	stopped   chan struct{}
	waitGroup sync.WaitGroup

	lock    util.GuardedLock
	result  mo.Option[ScrapeResult]
	waiters []chan<- ScrapeResult
}

func newScraper(feed feed.Feed, metrics observers) *Scraper {
	return &Scraper{
		feed:    feed,
		metrics: metrics,

		force:   make(chan struct{}, 1),
		stopped: make(chan struct{}),
	}
}

func (s *Scraper) start(ctx context.Context, develMode bool) {
	s.metrics.startTime().SetToCurrentTime()
	s.waitGroup.Go(func() {
		s.daemon(ctx, develMode)
	})
}

func (s *Scraper) stop(ctx context.Context) {
	logging.L(ctx).Infof("Stopping %q scrapper...", s.feed.Name())
	close(s.stopped)
	s.waitGroup.Wait()
	logging.L(ctx).Infof("%q scrapper has stopped.", s.feed.Name())
}

func (s *Scraper) Get(ctx context.Context) ScrapeResult {
	lock := s.lock.Lock()
	defer lock.UnlockIfLocked()

	if result, ok := s.result.Get(); ok {
		return result
	}

	waiter := make(chan ScrapeResult, 1)
	s.waiters = append(s.waiters, waiter)
	lock.Unlock()

	select {
	case s.force <- struct{}{}:
	default:
	}

	select {
	case result := <-waiter:
		return result

	case <-s.stopped:
		return makeErrorResult(http.StatusServiceUnavailable)

	case <-ctx.Done():
		lock.Lock()
		if index := slices.Index(s.waiters, waiter); index != -1 {
			s.waiters = slices.Delete(s.waiters, index, index+1)
		}
		return makeErrorResult(http.StatusGatewayTimeout)
	}
}

func (s *Scraper) daemon(ctx context.Context, develMode bool) {
	forceChan := s.force
	infiniteChan := make(chan struct{})

	updateTimer := time.NewTimer(rand.N(scrapePeriod))
	defer updateTimer.Stop()

	updateChan := updateTimer.C
	if develMode {
		updateChan = make(chan time.Time)
	}

	for {
		select {
		case <-updateChan:

		case <-forceChan:
			updateTimer.Stop()
			select {
			case <-updateTimer.C:
			default:
			}

		case <-s.stopped:
			return
		}

		result := s.scrape(ctx)

		lock := s.lock.Lock()
		s.result = mo.Some(result)
		waiters := s.waiters
		s.waiters = nil
		lock.Unlock()

		for _, waiter := range waiters {
			waiter <- result
		}

		updateTimer.Reset(scrapePeriod)
		forceChan = infiniteChan
	}
}

func (s *Scraper) scrape(ctx context.Context) ScrapeResult {
	ctx = fetch.WithContext(ctx, s.metrics.fetchDuration)

	logging.L(ctx).Infof("Scraping %q feed...", s.feed.Name())

	startTime := time.Now()
	feed, err := s.feed.Get(ctx)
	s.metrics.scrapeDuration.Observe(time.Since(startTime).Seconds())

	if err == nil {
		logging.L(ctx).Infof("%q feed scraped.", s.feed.Name())
		feed.Normalize()

		if data, err := rss.Generate(feed); err == nil {
			s.metrics.feedTime().SetToCurrentTime()
			s.metrics.feedStatus.WithLabelValues("success").Inc()
			return makeScrapeResult(http.StatusOK, rss.ContentType, data)
		} else {
			logging.L(ctx).Errorf("Failed to render %s RSS feed: %s.", s.feed.Name(), err)
			s.metrics.feedStatus.WithLabelValues("error").Inc()
			return makeErrorResult(http.StatusInternalServerError)
		}
	} else if util.IsTemporaryError(err) {
		logging.L(ctx).Warnf("Failed to scrape %q feed: %s.", s.feed.Name(), err)
		s.metrics.feedStatus.WithLabelValues("unavailable").Inc()
		return makeErrorResult(http.StatusGatewayTimeout)
	} else {
		logging.L(ctx).Errorf("Failed to scrape %q feed: %s.", s.feed.Name(), err)
		s.metrics.feedStatus.WithLabelValues("error").Inc()
		return makeErrorResult(http.StatusBadGateway)
	}
}

type ScrapeResult struct {
	HTTPStatus  int
	ContentType string
	Data        []byte
}

func makeScrapeResult(status int, contentType string, data []byte) ScrapeResult {
	return ScrapeResult{
		HTTPStatus:  status,
		ContentType: contentType,
		Data:        data,
	}
}

func makeErrorResult(status int) ScrapeResult {
	return makeScrapeResult(status, "text/plain", []byte("Failed to generate the RSS feed"))
}
