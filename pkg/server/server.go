package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/ggicci/httpin"
	"github.com/ggicci/httpin/integration"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/KonishchevDmitry/feedsd/internal/scraper"
	"github.com/KonishchevDmitry/feedsd/pkg/feed"
)

func init() {
	integration.UseGorillaMux("path", mux.Vars)
}

type Server struct {
	router   *mux.Router
	scrapers *scraper.Registry
}

func New() *Server {
	s := &Server{
		router:   mux.NewRouter(),
		scrapers: scraper.NewRegistry(),
	}
	s.register("/", func(ctx context.Context, writer http.ResponseWriter, request *http.Request) {
		http.NotFound(writer, request)
	})
	return s
}

func (s *Server) Register(feed feed.Feed) error {
	scraper, err := s.scrapers.Add(feed)
	if err != nil {
		return err
	}

	s.register(fmt.Sprintf("/%s.rss", feed.Name()), func(ctx context.Context, writer http.ResponseWriter, request *http.Request) {
		result := scraper.Get(ctx)
		result.Write(writer)
	})

	return nil
}

func RegisterParametrized[P feed.Params](s *Server, feed feed.ParametrizedFeed[P]) error {
	scraper, err := scraper.AddParametrized(s.scrapers, feed)
	if err != nil {
		return err
	}

	var path string
	if subPath, ok := feed.Path(); ok {
		path = fmt.Sprintf("/%s/%s", feed.Name(), strings.TrimPrefix(subPath, "/"))
	} else {
		path = fmt.Sprintf("/%s.rss", feed.Name())
	}

	s.register(path, func(ctx context.Context, writer http.ResponseWriter, request *http.Request) {
		params, err := httpin.Decode[P](request)
		if err != nil {
			logging.L(ctx).Warnf("Invalid feed parameters: %s.", err)
			http.NotFound(writer, request)
			return
		}

		// FIXME(konishchev): Context cancellation + locks
		result := scraper.Scrape(ctx, *params)
		result.Write(writer)
	})

	return nil
}

func (s *Server) Serve(ctx context.Context, feedsAddr string, metricsAddr string, develMode bool) error {
	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	if err := prometheus.DefaultRegisterer.Register(s.scrapers); err != nil {
		return err
	}

	//nolint:gosec
	feedsServer := http.Server{
		Addr:     feedsAddr,
		Handler:  s.router,
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
		if err := feedsServer.Serve(feedsSocket); !errors.Is(err, http.ErrServerClosed) {
			serverCrashed <- fmt.Errorf("feeds HTTP server has crashed: %w", err)
		}
	})

	closeMetricsSocket = false
	waitGroup.Go(func() {
		if err := metricsServer.Serve(metricsSocket); !errors.Is(err, http.ErrServerClosed) {
			serverCrashed <- fmt.Errorf("metrics HTTP server has crashed: %w", err)
		}
	})

	s.scrapers.Start(ctx, develMode)
	defer s.scrapers.Stop(ctx)

	return <-serverCrashed
}

func (s *Server) register(path string, handler func(ctx context.Context, writer http.ResponseWriter, request *http.Request)) {
	s.router.HandleFunc(path, func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logging.L(ctx).Debugf("%s %s...", request.Method, request.RequestURI)
		handler(ctx, writer, request)
		logging.L(ctx).Debugf("%s %s finished.", request.Method, request.RequestURI)
	})
}
