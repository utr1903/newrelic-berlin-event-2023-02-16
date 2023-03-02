package main

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

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

	err := performPreprocessing(r, &parentSpan)
	if err != nil {
		createHttpResponse(&w, http.StatusBadRequest, []byte("Fail"), &parentSpan)
		return
	}

	// Perform request to Donald service
	err = performRequestToDonald(w, r, &parentSpan)
	if err != nil {
		createHttpResponse(&w, http.StatusInternalServerError, []byte("Fail"), &parentSpan)
		return
	}

	createHttpResponse(&w, http.StatusOK, []byte("Success"), &parentSpan)
}

func performRequestToDonald(
	w http.ResponseWriter,
	r *http.Request,
	parentSpan *trace.Span,
) error {
	return performHttpCall(r.Method)
}

func performPreprocessing(
	r *http.Request,
	parentSpan *trace.Span,
) error {

	if considerPreprocessingSpans {
		_, processingSpan := (*parentSpan).TracerProvider().
			Tracer(appName).
			Start(
				r.Context(),
				"preprocessing",
				trace.WithSpanKind(trace.SpanKindInternal),
			)
		defer processingSpan.End()

		msg, err := produceException(r)
		if err != nil {
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
		return nil
	}

	_, err := produceException(r)
	return err
}

func produceException(
	r *http.Request,
) (
	string,
	error,
) {
	preprocessingException := r.URL.Query().Get("preprocessingException")
	if preprocessingException == "true" {
		msg := "Provided data format is invalid and cannot be processed"
		fmt.Println(msg)
		return msg, errors.New("preprocessing failed")
	}
	return "", nil
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
