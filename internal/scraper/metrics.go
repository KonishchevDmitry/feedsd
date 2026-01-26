package scraper

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	feedStatusSuccess     = "success"
	feedStatusUnavailable = "unavailable"
	feedStatusError       = "error"
	feedStatusPanic       = "panic"
)

type metrics struct {
	startTime      *prometheus.GaugeVec
	feedTime       *prometheus.GaugeVec
	feedStatus     *prometheus.CounterVec
	fetchDuration  *prometheus.HistogramVec
	scrapeDuration *prometheus.HistogramVec
}

func makeMetrics() metrics {
	return metrics{
		startTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "feeds_start_time",
			Help: "Scraper daemon start time",
		}, []string{"name"}),

		feedTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "feeds_time",
			Help: "Feed time",
		}, []string{"name"}),

		feedStatus: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "feeds_status_total",
			Help: "Feed generation status",
		}, []string{"name", "status"}),

		fetchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "feeds_fetch_duration",
			Help:    "Document fetch duration",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30, 40, 50, 60, 90},
		}, []string{"name"}),

		scrapeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "feeds_scrape_duration",
			Help:    "Feed scrape duration",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 900, 1200},
		}, []string{"name"}),
	}
}

type baseObservers struct {
	feedStatus     *prometheus.CounterVec
	fetchDuration  prometheus.Observer
	scrapeDuration prometheus.Observer
}

func (m *metrics) baseObservers(name string) baseObservers {
	return baseObservers{
		feedStatus:     m.feedStatus.MustCurryWith(prometheus.Labels{"name": name}),
		fetchDuration:  m.fetchDuration.WithLabelValues(name),
		scrapeDuration: m.scrapeDuration.WithLabelValues(name),
	}
}

type backgroundObservers struct {
	startTime func() prometheus.Gauge
	feedTime  func() prometheus.Gauge
}

func (m *metrics) backgroundObservers(name string) backgroundObservers {
	return backgroundObservers{
		startTime: func() prometheus.Gauge {
			return m.startTime.WithLabelValues(name)
		},
		feedTime: func() prometheus.Gauge {
			return m.feedTime.WithLabelValues(name)
		},
	}
}

var _ prometheus.Collector = &metrics{}

func (m *metrics) Describe(descs chan<- *prometheus.Desc) {
	m.startTime.Describe(descs)
	m.feedTime.Describe(descs)
	m.feedStatus.Describe(descs)
	m.fetchDuration.Describe(descs)
	m.scrapeDuration.Describe(descs)
}

func (m *metrics) Collect(metrics chan<- prometheus.Metric) {
	m.startTime.Collect(metrics)
	m.feedTime.Collect(metrics)
	m.feedStatus.Collect(metrics)
	m.fetchDuration.Collect(metrics)
	m.scrapeDuration.Collect(metrics)
}
