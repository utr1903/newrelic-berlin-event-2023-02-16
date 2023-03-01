package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	appName string

	donaldRequestInterval string
	donaldEndpoint        string
	donaldPort            string
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()
}

func parseFlags() {
	appName = os.Getenv("APP_NAME")
	donaldRequestInterval = os.Getenv("DONALD_REQUEST_INTERVAL")
	donaldEndpoint = os.Getenv("DONALD_ENDPOINT")
	donaldPort = os.Getenv("DONALD_PORT")
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
			httpList()
		}
	}()

	// DELETE simulator
	go func() {
		for {

			// Make request after each interval * 4
			time.Sleep(4 * time.Duration(interval) * time.Millisecond)

			// Delete
			httpDelete()
		}
	}()
}
