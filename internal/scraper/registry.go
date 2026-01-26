package scraper

import (
	"context"
	"fmt"
	"sync"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
)

type Registry struct {
	// FIXME(konishchev): Add simple scrapers
	scrapers map[string]*BackgroundScraper
	metrics
}

func NewRegistry() *Registry {
	return &Registry{
		scrapers: make(map[string]*BackgroundScraper),
		metrics:  makeMetrics(),
	}
}

func (r *Registry) Add(feed feed.Feed) (*BackgroundScraper, error) {
	name := feed.Name()
	if _, ok := r.scrapers[name]; ok {
		return nil, fmt.Errorf("%q feed is already registered", name)
	}

	scraper := newBackgroundScraper(feed, &r.metrics)
	r.scrapers[name] = scraper

	return scraper, nil
}

func (r *Registry) Start(ctx context.Context, develMode bool) {
	for _, scraper := range r.scrapers {
		scraper.start(ctx, develMode)
	}
}

func (r *Registry) Stop(ctx context.Context) {
	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	for _, scraper := range r.scrapers {
		waitGroup.Go(func() {
			scraper.stop(ctx)
		})
	}
}
