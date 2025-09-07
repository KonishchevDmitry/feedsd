package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"go.uber.org/zap/zapcore"

	"github.com/KonishchevDmitry/newslib/internal/util"
	"github.com/KonishchevDmitry/newslib/pkg/rss"
)

var mux = http.NewServeMux()

func init() {
	register("/", http.NotFound)
}

func Serve(ctx context.Context, addressPort string) error {
	errorLog := log.New(newHTTPLogger(logging.L(ctx)), "HTTP server: ", 0)

	server := http.Server{
		Addr:     addressPort,
		Handler:  mux,
		ErrorLog: errorLog,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	logging.L(ctx).Infof("Listening on %s...", addressPort)
	return server.ListenAndServe()
}

func Register(path string, generator func(context.Context) (*rss.Feed, error)) {
	register(path, func(w http.ResponseWriter, r *http.Request) {
		generate(w, r, generator)
	})
}

func register(path string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logging.L(ctx).Infof("%s %s...", r.Method, r.RequestURI)
		handler(w, r)
		logging.L(ctx).Infof("%s %s finished.", r.Method, r.RequestURI)
	})
}

func generate(w http.ResponseWriter, r *http.Request, generator func(context.Context) (*rss.Feed, error)) {
	ctx := r.Context()

	feed, err := generator(ctx)
	if err != nil {
		status, level := http.StatusBadGateway, zapcore.ErrorLevel
		for err := err; err != nil; err = errors.Unwrap(err) {
			if err, ok := err.(util.Temporary); ok && err.Temporary() {
				status, level = http.StatusGatewayTimeout, zapcore.WarnLevel
				break
			}
		}

		logging.L(ctx).Logf(level, "Failed to generate %s RSS feed: %s.", r.RequestURI, err)
		writeError(w, status)
		return
	}

	data, err := rss.Generate(feed, true)
	if err != nil {
		logging.L(ctx).Errorf("Failed to render %s RSS feed: %s.", r.RequestURI, err)
		writeError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, "Failed to generate the RSS feed", status)
}
