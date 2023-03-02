package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	appName string
	appPort string

	donaldRequestInterval string
	donaldEndpoint        string
	donaldPort            string

	considerPreprocessingSpans bool
)

func main() {

	// Parse arguments and feature flags
	parseFlags()

	// Get context
	ctx := context.Background()

	// Create tracer provider
	tp := newTraceProvider(ctx)
	defer shutdownTraceProvider(ctx, tp)

	// Create metric provider
	mp := newMetricProvider(ctx)
	defer shutdownMetricProvider(ctx, mp)

	// Simulate
	go simulate()

	// Serve
	http.Handle("/api", otelhttp.NewHandler(http.HandlerFunc(handler), "api"))
	http.ListenAndServe(":"+appPort, nil)
}

func parseFlags() {
	appName = os.Getenv("APP_NAME")
	appPort = os.Getenv("APP_PORT")

	donaldRequestInterval = os.Getenv("DONALD_REQUEST_INTERVAL")
	donaldEndpoint = os.Getenv("DONALD_ENDPOINT")
	donaldPort = os.Getenv("DONALD_PORT")

	considerPreprocessingSpans, _ = strconv.ParseBool(os.Getenv("CONSIDER_PREPROCESSING_SPANS"))
}

func simulate() {

	interval, err := strconv.ParseInt(donaldRequestInterval, 10, 64)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	httpClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   time.Duration(30 * time.Second),
	}

	httpClientDuration, err = global.MeterProvider().
		Meter(appName).
		Float64Histogram("http.client.duration")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// LIST simulator
	go func() {
		for {

			// Make request after each interval
			time.Sleep(time.Duration(interval) * time.Millisecond)

			// List
			performHttpCall(http.MethodGet)
		}
	}()

	// DELETE simulator
	go func() {
		for {

			// Make request after each interval * 4
			time.Sleep(4 * time.Duration(interval) * time.Millisecond)

			// Delete
			performHttpCall(http.MethodDelete)
		}
	}()
}
