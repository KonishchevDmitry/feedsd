package cache

import (
	"context"
	"net/url"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	cache "github.com/go-pkgz/expirable-cache/v3"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

type Cache[T any] struct {
	cache cache.Cache[string, T]
}

func New[T any]() *Cache[T] {
	return &Cache[T]{
		cache: cache.NewCache[string, T](),
	}
}

func (c *Cache[T]) Cached(
	ctx context.Context, url *url.URL,
	fetch func(ctx context.Context, url *url.URL) (T, error),
) (T, error) {
	key := url.String()

	if value, ok := c.cache.Get(key); ok {
		logging.L(ctx).Debugf("Got %s from cache.", url)
		return value, nil
	}

	value, err := fetch(ctx, url)
	if err == nil {
		logging.L(ctx).Debugf("Add %s to cache.", url)
		c.cache.Add(key, value)
	}

	return value, err
}

func (c *Cache[T]) PopulateFeed(
	ctx context.Context, feed *rss.Feed,
	fetch func(ctx context.Context, url *url.URL) (T, error),
	apply func(details T, item *rss.Item),
) error {
	c.Cleanup(ctx, feed)

	for _, item := range feed.Items {
		url, err := url.Parse(item.Link)
		if err != nil {
			return err
		}

		value, err := c.Cached(ctx, url, fetch)
		if err != nil {
			return err
		}

		apply(value, item)
	}

	return nil
}

func (c *Cache[T]) Cleanup(ctx context.Context, feed *rss.Feed) {
	urls := make(map[string]struct{})
	for _, item := range feed.Items {
		urls[item.Link] = struct{}{}
	}

	c.cache.InvalidateFn(func(url string) bool {
		if _, ok := urls[url]; ok {
			return false
		}

		logging.L(ctx).Debugf("Drop %s from cache.", url)
		return true
	})
}
