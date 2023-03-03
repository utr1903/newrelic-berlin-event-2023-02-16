package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

func initLogger() {

	// Set log level
	switch logLevel {
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

func log(
	lvl logrus.Level,
	ctx context.Context,
	user string,
	msg string,
) {
	span := trace.SpanFromContext(ctx)
	if logWithContext && span.SpanContext().HasTraceID() && span.SpanContext().HasSpanID() {
		logrus.WithFields(logrus.Fields{
			"service.name": appName,
			"trace.id":     span.SpanContext().TraceID().String(),
			"span.id":      span.SpanContext().SpanID().String(),
		}).Error("user:" + user + "|" + msg)
	} else {
		logrus.Error("user:" + user + "|" + msg)
	}
}
