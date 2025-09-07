package server

import (
	"io"

	"go.uber.org/zap"
)

type httpLogger struct {
	logger *zap.SugaredLogger
}

func newHTTPLogger(logger *zap.SugaredLogger) *httpLogger {
	return &httpLogger{logger}
}

var _ io.Writer = &httpLogger{}

func (l *httpLogger) Write(p []byte) (n int, err error) {
	l.logger.Errorf("%s.", p)
	return len(p), nil
}
