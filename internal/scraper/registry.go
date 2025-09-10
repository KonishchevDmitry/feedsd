package scraper

import (
	"context"
	"fmt"
	"sync"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/prometheus/client_golang/prometheus"
)

var Registry = make(ScraperRegistry)

type ScraperRegistry map[string]*Scraper

func (r ScraperRegistry) Add(feed feed.Feed) (*Scraper, error) {
	name := feed.Name()
	if _, ok := r[name]; ok {
		return nil, fmt.Errorf("%q feed is already registered", name)
	}

	scraper := newScraper(feed)
	r[name] = scraper

	return scraper, nil
}

func (r ScraperRegistry) Start(ctx context.Context, develMode bool) {
	for _, scraper := range r {
		scraper.start(ctx, develMode)
	}
}

func (r ScraperRegistry) Stop(ctx context.Context) {
	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	for _, scraper := range r {
		waitGroup.Go(func() {
			scraper.stop(ctx)
		})
	}
}

var _ prometheus.Collector = ScraperRegistry{}

func (r ScraperRegistry) Describe(descs chan<- *prometheus.Desc) {
	descs <- feedAgeMetric
	descs <- feedStatusMetric
}

func (r ScraperRegistry) Collect(metrics chan<- prometheus.Metric) {
	for _, scraper := range r {
		scraper.Collect(metrics)
	}
}
