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

type SimpleScraper struct {
	baseScraper
}

func newSimpleScraper(feed feed.Feed, metrics *baseObservers) *SimpleScraper {
	return &SimpleScraper{
		baseScraper: makeBaseScraper(feed, metrics),
	}
}

func (s *SimpleScraper) Scrape(ctx context.Context) ScrapeResult {
	return s.scrape(ctx)
}

type SimpleParametrizedScraper[P feed.Params] struct {
	feed    feed.ParametrizedFeed[P]
	metrics *baseObservers
}

func newSimpleParametrizedScraper[P feed.Params](feed feed.ParametrizedFeed[P], metrics *baseObservers) *SimpleParametrizedScraper[P] {
	return &SimpleParametrizedScraper[P]{
		feed:    feed,
		metrics: metrics,
	}
}

func (s *SimpleParametrizedScraper[P]) Scrape(ctx context.Context, params P) ScrapeResult {
	// Attention: Binding changes feed name, so be careful and construct metric observers before the binding
	boundFeed := feed.BindParams(s.feed, params)
	return newSimpleScraper(boundFeed, s.metrics).scrape(ctx)
}

type BackgroundScraper struct {
	baseScraper
	backgroundMetrics *backgroundObservers

	force     chan struct{}
	stopped   chan struct{}
	waitGroup sync.WaitGroup

	lock    util.GuardedLock
	result  mo.Option[ScrapeResult]
	waiters []chan<- ScrapeResult
}

func newBackgroundScraper(feed feed.Feed, baseMetrics *baseObservers, backgroundMetrics *backgroundObservers) *BackgroundScraper {
	return &BackgroundScraper{
		baseScraper:       makeBaseScraper(feed, baseMetrics),
		backgroundMetrics: backgroundMetrics,

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
	logging.L(ctx).Infof("Stopping %s scrapper...", s.feed.Name())
	close(s.stopped)
	s.waitGroup.Wait()
	logging.L(ctx).Infof("%s scrapper has stopped.", s.feed.Name())
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
	baseMetrics *baseObservers
}

func makeBaseScraper(feed feed.Feed, metrics *baseObservers) baseScraper {
	return baseScraper{
		feed:        feed,
		baseMetrics: metrics,
	}
}

func (s *baseScraper) scrape(ctx context.Context) ScrapeResult {
	ctx = fetch.WithContext(ctx, s.baseMetrics.fetchDuration)
	logging.L(ctx).Infof("Scraping %s feed...", s.feed.Name())

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
		logging.L(ctx).Errorf("Failed to scrape %s feed: %s", s.feed.Name(), panicErr)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusPanic).Inc()
		return makeErrorResult(http.StatusInternalServerError)
	} else if util.IsTemporaryError(err) {
		logging.L(ctx).Warnf("Failed to scrape %s feed: %s.", s.feed.Name(), err)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusUnavailable).Inc()
		return makeErrorResult(http.StatusGatewayTimeout)
	} else if err != nil {
		logging.L(ctx).Errorf("Failed to scrape %s feed: %s.", s.feed.Name(), err)
		s.baseMetrics.feedStatus.WithLabelValues(feedStatusError).Inc()
		return makeErrorResult(http.StatusBadGateway)
	}

	logging.L(ctx).Infof("%s feed scraped.", s.feed.Name())
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

func (r *ScrapeResult) Write(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", r.ContentType)
	writer.WriteHeader(r.HTTPStatus)
	_, _ = writer.Write(r.Data)
}
