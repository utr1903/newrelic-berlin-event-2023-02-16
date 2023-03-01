package main

import (
	"context"
	"os"
	"os/signal"
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
	tp := NewTraceProvider(ctx)
	defer ShutdownTraceProvider(ctx, tp)

	// Create metric provider
	mp := NewMetricProvider(ctx)
	defer ShutdownMetricProvider(ctx, mp)

	// Simulate
	go SimulateHttpServer()

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
