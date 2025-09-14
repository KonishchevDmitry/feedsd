package scraper

import (
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	startTime prometheus.Gauge

	feedTime       *prometheus.GaugeVec
	feedStatus     *prometheus.CounterVec
	fetchDuration  *prometheus.HistogramVec
	scrapeDuration *prometheus.HistogramVec
}

func makeMetrics() metrics {
	startTime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "feeds_start_time",
		Help:        "Daemon start time",
		ConstLabels: prometheus.Labels{},
	})
	startTime.SetToCurrentTime()

	return metrics{
		startTime: startTime,

		feedTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "feeds_time",
			Help: "Feed time",
		}, []string{"name"}),

		feedStatus: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "feeds_status",
			Help: "Feed generation status",
		}, []string{"name", "status"}),

		// FIXME(konishchev): Implement
		fetchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "feeds_fetch_duration",
			Help:    "Document fetch duration",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		}, []string{"name"}),

		scrapeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "feeds_scrape_duration",
			Help:    "Feed scrape duration",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600},
		}, []string{"name"}),
	}
}

type observers struct {
	feedTime       prometheus.Gauge
	feedStatus     *prometheus.CounterVec
	fetchDuration  prometheus.Observer
	scrapeDuration prometheus.Observer
}

func (m *metrics) observers(name string) observers {
	return observers{
		feedTime:       m.feedTime.WithLabelValues(name),
		feedStatus:     m.feedStatus.MustCurryWith(prometheus.Labels{"name": name}),
		fetchDuration:  m.fetchDuration.WithLabelValues(name),
		scrapeDuration: m.scrapeDuration.WithLabelValues(name),
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
