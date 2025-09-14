package scraper

import (
	"context"
	"fmt"
	"sync"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
)

type Registry struct {
	scrapers map[string]*Scraper
	metrics
}

func NewRegistry() *Registry {
	return &Registry{
		scrapers: make(map[string]*Scraper),
		metrics:  makeMetrics(),
	}
}

func (r *Registry) Add(feed feed.Feed) (*Scraper, error) {
	name := feed.Name()
	if _, ok := r.scrapers[name]; ok {
		return nil, fmt.Errorf("%q feed is already registered", name)
	}

	scraper := newScraper(feed, r.metrics.observers(name))
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
