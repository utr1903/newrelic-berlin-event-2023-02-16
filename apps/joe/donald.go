package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	httpClient         *http.Client
	httpClientDuration instrument.Float64Histogram
)

func performHttpCall(
	httpMethod string,
) error {

	// Get context
	ctx := context.Background()

	// Create request propagation
	carrier := propagation.HeaderCarrier(http.Header{})
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Create HTTP request with trace context
	req, err := http.NewRequestWithContext(
		ctx, httpMethod,
		"http://"+donaldEndpoint+":"+donaldPort+"/api",
		nil,
	)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")

	// Start timer
	requestStartTime := time.Now()

	// Perform HTTP request
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		recordClientDuration(ctx, httpMethod, http.StatusInternalServerError, requestStartTime)
		return err
	}
	defer res.Body.Close()

	// Read HTTP response
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		recordClientDuration(ctx, httpMethod, res.StatusCode, requestStartTime)
		return err
	}

	recordClientDuration(ctx, httpMethod, res.StatusCode, requestStartTime)
	return nil
}

func recordClientDuration(
	ctx context.Context,
	httpMethod string,
	statusCode int,
	startTime time.Time,
) {
	elapsedTime := float64(time.Since(startTime)) / float64(time.Millisecond)
	httpserverPortAsInt, _ := strconv.Atoi(donaldPort)
	attributes := attribute.NewSet(
		semconv.HTTPSchemeHTTP,
		semconv.HTTPFlavorKey.String("1.1"),
		semconv.HTTPMethod(httpMethod),
		semconv.NetPeerName(donaldEndpoint),
		semconv.NetPeerPort(httpserverPortAsInt),
		semconv.HTTPStatusCode(statusCode),
	)

	httpClientDuration.Record(ctx, elapsedTime, attributes.ToSlice()...)
}
