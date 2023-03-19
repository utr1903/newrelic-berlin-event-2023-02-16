package main

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func handler(
	w http.ResponseWriter,
	r *http.Request,
) {

	// Get server span
	parentSpan := trace.SpanFromContext(r.Context())
	defer parentSpan.End()

	// Get caller user
	user := r.Header.Get("X-User-ID")
	if user == "" {
		user = "_anonymous_"
	}

	log(logrus.InfoLevel, r.Context(), user, "Handler is triggered")

	err := performPreprocessing(r, &parentSpan, user)
	if err != nil {
		createHttpResponse(&w, http.StatusBadRequest, []byte("Fail"), &parentSpan)
		return
	}

	// Perform request to Donald service
	err = performRequestToDonald(r, user)
	if err != nil {
		createHttpResponse(&w, http.StatusInternalServerError, []byte("Fail"), &parentSpan)
		return
	}

	createHttpResponse(&w, http.StatusOK, []byte("Success"), &parentSpan)
}

func performRequestToDonald(
	r *http.Request,
	user string,
) error {
	// Add request parameters
	reqParams := map[string]string{}
	for k, v := range r.URL.Query() {
		reqParams[k] = v[0]
	}
	// Make the call
	return performHttpCall(r.Context(), r.Method, user, reqParams)
}

func performPreprocessing(
	r *http.Request,
	parentSpan *trace.Span,
	user string,
) error {

	log(logrus.InfoLevel, r.Context(), user, "Preprocessing...")
	if considerPreprocessingSpans {
		ctx, processingSpan := (*parentSpan).TracerProvider().
			Tracer(appName).
			Start(
				r.Context(),
				"preprocessing",
				trace.WithSpanKind(trace.SpanKindInternal),
			)
		defer processingSpan.End()

		err := produceException(r)
		if err != nil {

			msg := "Provided data format is invalid and cannot be processed."
			log(logrus.ErrorLevel, ctx, user, msg)

			stackSlice := make([]byte, 512)
			s := runtime.Stack(stackSlice, false)

			attrs := []attribute.KeyValue{
				attribute.String("exception.type", "joe.preprocessing"),
				attribute.String("exception.message", msg),
				attribute.String("exception.stacktrace", string(stackSlice[0:s])),
			}
			processingSpan.SetAttributes(attrs...)
			return err
		}
		log(logrus.InfoLevel, r.Context(), user, "Preprocessing is completed.")
		return nil
	}

	err := produceException(r)
	return err
}

func produceException(
	r *http.Request,
) error {
	preprocessingException := r.URL.Query().Get("preprocessingException")
	if preprocessingException == "true" {
		return errors.New("preprocessing failed")
	}
	return nil
}

func createHttpResponse(
	w *http.ResponseWriter,
	statusCode int,
	body []byte,
	serverSpan *trace.Span,
) {
	(*w).WriteHeader(statusCode)
	(*w).Write(body)

	attrs := []attribute.KeyValue{
		semconv.HTTPStatusCode(statusCode),
	}
	(*serverSpan).SetAttributes(attrs...)
}
