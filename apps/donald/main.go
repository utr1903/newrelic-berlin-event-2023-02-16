package main

import (
	"context"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	appName                     string
	appPort                     string
	considerDatabaseSpans       bool
	considerPostprocessingSpans bool
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

	// Connect to MySQL
	db = createDatabaseConnection()
	defer db.Close()

	// Serve
	http.Handle("/api", otelhttp.NewHandler(http.HandlerFunc(handler), "delete"))
	http.ListenAndServe(":"+appPort, nil)
}

func parseFlags() {
	appName = os.Getenv("APP_NAME")
	appPort = os.Getenv("APP_PORT")
	considerDatabaseSpans, _ = strconv.ParseBool(os.Getenv("CONSIDER_DATABASE_SPANS"))
	considerPostprocessingSpans, _ = strconv.ParseBool(os.Getenv("CONSIDER_POSTPROCESSING_SPANS"))
}
