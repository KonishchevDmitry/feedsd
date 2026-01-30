package scraper

import (
	"context"
	"fmt"
	"sync"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
)

type Registry struct {
	scrapers           map[string]struct{}
	backgroundScrapers []*BackgroundScraper
	metrics
}

func NewRegistry() *Registry {
	return &Registry{
		scrapers: make(map[string]struct{}),
		metrics:  makeMetrics(),
	}
}

func (r *Registry) Add(feed feed.Feed) (*BackgroundScraper, error) {
	name := feed.Name()
	if err := r.add(name); err != nil {
		return nil, err
	}

	scraper := newBackgroundScraper(feed, r.metrics.baseObservers(name), r.metrics.backgroundObservers(name))
	r.backgroundScrapers = append(r.backgroundScrapers, scraper)

	return scraper, nil
}

func AddParametrized[P feed.Params](r *Registry, feed feed.ParametrizedFeed[P]) (*SimpleParametrizedScraper[P], error) {
	name := feed.Name()
	if err := r.add(name); err != nil {
		return nil, err
	}

	scraper := newSimpleParametrizedScraper(feed, r.metrics.baseObservers(name))
	return scraper, nil
}

func (r *Registry) add(name string) error {
	if _, ok := r.scrapers[name]; ok {
		return fmt.Errorf("%q feed is already registered", name)
	}
	r.scrapers[name] = struct{}{}
	return nil
}

func (r *Registry) Start(ctx context.Context, develMode bool) {
	for _, scraper := range r.backgroundScrapers {
		scraper.start(ctx, develMode)
	}
}

func (r *Registry) Stop(ctx context.Context) {
	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	for _, scraper := range r.backgroundScrapers {
		waitGroup.Go(func() {
			scraper.stop(ctx)
		})
	}
}
