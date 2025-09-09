package scraper

import "github.com/prometheus/client_golang/prometheus"

var (
	feedAgeMetric = prometheus.NewDesc(
		"news_feed_age", "Feed age", []string{"name"}, nil)

	feedStatusMetric = prometheus.NewDesc(
		"news_feed_status", "Feed generation status", []string{"name", "status"}, nil)
)
