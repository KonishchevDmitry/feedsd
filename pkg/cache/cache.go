package cache

import (
	"context"
	"net/url"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/go-pkgz/expirable-cache/v3"
)

type Cache[T any] struct {
	cache cache.Cache[string, T]
}

func New[T any]() *Cache[T] {
	return &Cache[T]{
		cache: cache.NewCache[string, T](),
	}
}

func (c *Cache[T]) PopulateFeed(
	ctx context.Context, feed *rss.Feed,
	fetch func(ctx context.Context, url *url.URL) (T, error),
	apply func(details T, item *rss.Item),
) error {
	urls := make(map[string]struct{})
	for _, item := range feed.Items {
		urls[item.Link] = struct{}{}
	}

	c.cache.InvalidateFn(func(url string) bool {
		if _, ok := urls[url]; ok {
			return false
		}

		logging.L(ctx).Debugf("Drop %s details from cache.", url)
		return true
	})

	for _, item := range feed.Items {
		details, ok := c.cache.Get(item.Link)
		if ok {
			logging.L(ctx).Debugf("Got %s details from cache.", item.Link)
		} else {
			url, err := url.Parse(item.Link)
			if err != nil {
				return err
			}

			details, err = fetch(ctx, url)
			if err != nil {
				return err
			}

			c.cache.Add(item.Link, details)
		}

		apply(details, item)
	}

	return nil
}
