package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {
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
