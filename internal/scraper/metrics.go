package scraper

import "github.com/prometheus/client_golang/prometheus"

var (
	feedAgeMetric = prometheus.NewDesc(
		"feeds_age", "Feed age", []string{"name"}, nil)

	feedStatusMetric = prometheus.NewDesc(
		"feeds_status", "Feed generation status", []string{"name", "status"}, nil)
)
