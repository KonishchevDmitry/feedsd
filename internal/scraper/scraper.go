package scraper

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"runtime/debug"
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

type BackgroundScraper struct {
	baseScraper
	backgroundMetrics backgroundObservers

	force     chan struct{}
	stopped   chan struct{}
	waitGroup sync.WaitGroup

	lock    util.GuardedLock
	result  mo.Option[ScrapeResult]
	waiters []chan<- ScrapeResult
}

func newBackgroundScraper(feed feed.Feed, metrics *metrics) *BackgroundScraper {
	return &BackgroundScraper{
		baseScraper:       makeBaseScraper(feed, metrics),
		backgroundMetrics: metrics.backgroundObservers(feed.Name()),

		force:   make(chan struct{}, 1),
		stopped: make(chan struct{}),
	}
}

func (s *BackgroundScraper) start(ctx context.Context, develMode bool) {
	s.backgroundMetrics.startTime().SetToCurrentTime()
	s.waitGroup.Go(func() {
		s.daemon(ctx, develMode)
	})
}

func (s *BackgroundScraper) stop(ctx context.Context) {
	logging.L(ctx).Infof("Stopping %q scrapper...", s.feed.Name())
	close(s.stopped)
	s.waitGroup.Wait()
	logging.L(ctx).Infof("%q scrapper has stopped.", s.feed.Name())
}

func (s *BackgroundScraper) Get(ctx context.Context) ScrapeResult {
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

func (s *BackgroundScraper) daemon(ctx context.Context, develMode bool) {
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
		if result.HTTPStatus == http.StatusOK {
			s.backgroundMetrics.feedTime().SetToCurrentTime()
		}

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

type baseScraper struct {
	feed        feed.Feed
	baseMetrics baseObservers
}

func makeBaseScraper(feed feed.Feed, metrics *metrics) baseScraper {
	return baseScraper{
		feed:        feed,
		baseMetrics: metrics.baseObservers(feed.Name()),
	}
}

func (s *baseScraper) scrape(ctx context.Context) ScrapeResult {
	ctx = fetch.WithContext(ctx, s.baseMetrics.fetchDuration)
	logging.L(ctx).Infof("Scraping %q feed...", s.feed.Name())

	var panicErr error
	startTime := time.Now()
	feed, err := func() (*rss.Feed, error) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				panicErr = fmt.Errorf("feed generator has panicked: %v\n%s", err, bytes.TrimRight(stack, "\n"))
			}
		}()
		return s.feed.Get(ctx)
	}()
	s.baseMetrics.scrapeDuration.Observe(time.Since(startTime).Seconds())

	if panicErr != nil {
		logging.L(ctx).Errorf("Failed to scrape %q feed: %s", s.feed.Name(), panicErr)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusPanic).Inc()
		return makeErrorResult(http.StatusInternalServerError)
	} else if util.IsTemporaryError(err) {
		logging.L(ctx).Warnf("Failed to scrape %q feed: %s.", s.feed.Name(), err)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusUnavailable).Inc()
		return makeErrorResult(http.StatusGatewayTimeout)
	} else if err != nil {
		logging.L(ctx).Errorf("Failed to scrape %q feed: %s.", s.feed.Name(), err)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusError).Inc()
		return makeErrorResult(http.StatusBadGateway)
	}

	logging.L(ctx).Infof("%q feed scraped.", s.feed.Name())
	feed.Normalize()

	data, err := rss.Generate(feed)
	if err != nil {
		logging.L(ctx).Errorf("Failed to render %s RSS feed: %s.", s.feed.Name(), err)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusError).Inc()
		return makeErrorResult(http.StatusInternalServerError)
	}

	s.baseMetrics.feedStatus.WithLabelValues(feedStatusSuccess).Inc()
	return makeScrapeResult(http.StatusOK, rss.ContentType, data)
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
