package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type httpLogger struct {
	logger *zap.SugaredLogger
}

func newHTTPLogger(logger *zap.SugaredLogger) *httpLogger {
	return &httpLogger{logger}
}

var _ io.Writer = &httpLogger{}

func (l *httpLogger) Write(message []byte) (int, error) {
	size := len(message)
	if size != 0 && message[size-1] == '\n' {
		message = message[:size-1]
	}

	l.logger.Errorf("%s.", message)
	return size, nil
}

type prometheusLogger struct {
	logger *zap.SugaredLogger
}

func newPrometheusLogger(logger *zap.SugaredLogger) *prometheusLogger {
	return &prometheusLogger{logger}
}

var _ promhttp.Logger = prometheusLogger{}

func (l prometheusLogger) Println(v ...any) {
	level := zapcore.ErrorLevel

	for _, value := range v {
		if err, ok := value.(error); ok {
			var netErr *net.OpError
			if errors.As(err, &netErr) && netErr.Op == "write" &&
				(netErr.Timeout() || errors.Is(netErr.Err, syscall.EPIPE)) || errors.Is(err, context.Canceled) {
				level = zap.DebugLevel
			}
			break
		}
	}

	l.logger.Logf(level, "Prometheus: %s.", strings.TrimRight(fmt.Sprintf("%v", v), "\n"))
}
