package collector

import (
	"context"
	"slices"
	"time"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/KonishchevDmitry/newslib/internal/util"
	newslib "github.com/KonishchevDmitry/newslib/pkg"
	"github.com/KonishchevDmitry/newslib/pkg/mo"
	"github.com/KonishchevDmitry/newslib/pkg/rss"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"
)

const collectionPeriod = time.Hour

type Collector struct {
	generator newslib.FeedGenerator
	started   chan struct{}
	stat      collectionStat

	lock    util.GuardedLock
	result  mo.Option[collectionResult]
	waiters []chan<- collectionResult
}

func NewCollector(generator newslib.FeedGenerator) *Collector {
	var stat collectionStat
	stat.feedTime.Store(time.Now())

	return &Collector{
		generator: generator,
		started:   make(chan struct{}, 1),
		stat:      stat,
	}
}

func (c *Collector) Start(ctx context.Context, develMode bool) {
	if !develMode {
		select {
		case c.started <- struct{}{}:
		default:
		}
	}
	go c.daemon(ctx)
}

func (c *Collector) Get(ctx context.Context) (*rss.Feed, error) {
	lock := c.lock.Lock()
	defer lock.UnlockIfLocked()

	if result, ok := c.result.Get(); ok {
		return result.feed, result.error
	}

	waiter := make(chan collectionResult, 1)
	c.waiters = append(c.waiters, waiter)
	lock.Unlock()

	select {
	case c.started <- struct{}{}:
	default:
	}

	select {
	case result := <-waiter:
		return result.feed, result.error

	case <-ctx.Done():
		lock.Lock()
		if index := slices.Index(c.waiters, waiter); index != -1 {
			c.waiters = slices.Delete(c.waiters, index, index+1)
		}
		return nil, ctx.Err()
	}
}

var _ prometheus.Collector = &Collector{}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
}

func (c *Collector) Collect(metrics chan<- prometheus.Metric) {
	metrics <- prometheus.MustNewConstMetric(
		feedAgeMetric, prometheus.GaugeValue,
		time.Since(c.stat.feedTime.Load()).Seconds(), c.generator.Name())

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.success.Load()), c.generator.Name(), "success")

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.unavailable.Load()), c.generator.Name(), "unavailable")

	metrics <- prometheus.MustNewConstMetric(
		feedStatusMetric, prometheus.CounterValue,
		float64(c.stat.error.Load()), c.generator.Name(), "error")
}

func (c *Collector) daemon(ctx context.Context) {
	<-c.started

	// FIXME(konishchev): Random delay?
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		<-timer.C

		logging.L(ctx).Infof("Collecting %q feed...", c.generator.Name())
		feed, err := c.generator.Get(ctx)
		result := collectionResult{
			feed:  feed,
			error: err,
		}
		if err == nil {
			logging.L(ctx).Infof("%q feed collected.", c.generator.Name())
			c.stat.success.Inc()
		} else if util.IsTemporaryError(err) {
			logging.L(ctx).Warnf("Failed to collect %q feed: %s.", c.generator.Name(), err)
			c.stat.unavailable.Inc()
		} else {
			logging.L(ctx).Errorf("Failed to collect %q feed: %s.", c.generator.Name(), err)
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

		timer.Reset(collectionPeriod)
	}
}

type collectionResult struct {
	feed  *rss.Feed
	error error
}

type collectionStat struct {
	feedTime    atomic.Time
	success     atomic.Int64
	unavailable atomic.Int64
	error       atomic.Int64
}
