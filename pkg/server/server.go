package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/KonishchevDmitry/feedsd/internal/scraper"
	"github.com/KonishchevDmitry/feedsd/pkg/feed"
)

var feedsMux = http.NewServeMux()

func init() {
	register("/", http.NotFound)
}

func Serve(ctx context.Context, feedsAddr string, metricsAddr string, develMode bool) error {
	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	if err := prometheus.DefaultRegisterer.Register(scraper.Registry); err != nil {
		return err
	}

	//nolint:gosec
	feedsServer := http.Server{
		Addr:     feedsAddr,
		Handler:  feedsMux,
		ErrorLog: log.New(newHTTPLogger(logging.L(ctx)), "Feeds HTTP server: ", 0),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	defer func() {
		if err := feedsServer.Shutdown(ctx); err != nil {
			logging.L(ctx).Errorf("Failed to shutdown feeds HTTP server: %s.", err)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog: newPrometheusLogger(logging.L(ctx)),
	}))

	//nolint:gosec
	metricsServer := http.Server{
		Addr:     metricsAddr,
		Handler:  metricsMux,
		ErrorLog: log.New(newHTTPLogger(logging.L(ctx)), "Metrics HTTP server: ", 0),
	}
	defer func() {
		if err := metricsServer.Shutdown(ctx); err != nil {
			logging.L(ctx).Errorf("Failed to shutdown metrics HTTP server: %s.", err)
		}
	}()

	logging.L(ctx).Infof("Listening on %s (feeds) and %s (metrics)...", feedsAddr, metricsAddr)

	feedsSocket, err := net.Listen("tcp", feedsAddr)
	if err != nil {
		return err
	}
	closeFeedsSocket := true
	defer func() {
		if closeFeedsSocket {
			if err := feedsSocket.Close(); err != nil {
				logging.L(ctx).Errorf("Failed to close a socket: %s.", err)
			}
		}
	}()

	metricsSocket, err := net.Listen("tcp", metricsAddr)
	if err != nil {
		return err
	}
	closeMetricsSocket := true
	defer func() {
		if closeMetricsSocket {
			if err := metricsSocket.Close(); err != nil {
				logging.L(ctx).Errorf("Failed to close a socket: %s.", err)
			}
		}
	}()

	serverCrashed := make(chan error, 2)

	closeFeedsSocket = false
	waitGroup.Go(func() {
		if err := feedsServer.Serve(feedsSocket); err != nil {
			serverCrashed <- fmt.Errorf("feeds HTTP server has crashed: %w", err)
		}
	})

	closeMetricsSocket = false
	waitGroup.Go(func() {
		if err := metricsServer.Serve(metricsSocket); err != nil {
			serverCrashed <- fmt.Errorf("metrics HTTP server has crashed: %w", err)
		}
	})

	scraper.Registry.Start(ctx, develMode)
	defer scraper.Registry.Stop(ctx)

	return <-serverCrashed
}

func Register(feed feed.Feed) {
	scraper, err := scraper.Registry.Add(feed)
	if err != nil {
		panic(err)
	}

	register(fmt.Sprintf("/%s.rss", feed.Name()), func(w http.ResponseWriter, r *http.Request) {
		response := scraper.Get(r.Context())
		w.Header().Set("Content-Type", response.ContentType)
		w.WriteHeader(response.HTTPStatus)
		_, _ = w.Write(response.Data)
	})
}

func register(path string, handler func(http.ResponseWriter, *http.Request)) {
	feedsMux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logging.L(ctx).Debugf("%s %s...", r.Method, r.RequestURI)
		handler(w, r)
		logging.L(ctx).Debugf("%s %s finished.", r.Method, r.RequestURI)
	})
}
