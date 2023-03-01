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

func httpList() {

	// Get context
	ctx := context.Background()

	// Create request propagation
	carrier := propagation.HeaderCarrier(http.Header{})
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Create HTTP request with trace context
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		"http://"+donaldEndpoint+":"+donaldPort+"/list",
		nil,
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")

	// Start timer
	requestStartTime := time.Now()

	// Perform HTTP request
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		recordClientDuration(ctx, requestStartTime, res.StatusCode)
		return
	}
	defer res.Body.Close()

	// Read HTTP response
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
	}

	recordClientDuration(ctx, requestStartTime, res.StatusCode)
}

func httpDelete() {

	// Get context
	ctx := context.Background()

	// Create request propagation
	carrier := propagation.HeaderCarrier(http.Header{})
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Create HTTP request with trace context
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		"http://"+donaldEndpoint+":"+donaldPort+"/delete",
		nil,
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")

	// Start timer
	requestStartTime := time.Now()

	// Perform HTTP request
	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		recordClientDuration(ctx, requestStartTime, res.StatusCode)
		return
	}
	defer res.Body.Close()

	// Read HTTP response
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
	}

	recordClientDuration(ctx, requestStartTime, res.StatusCode)
}

func recordClientDuration(
	ctx context.Context,
	startTime time.Time,
	statusCode int,
) {
	elapsedTime := float64(time.Since(startTime)) / float64(time.Millisecond)
	httpserverPortAsInt, _ := strconv.Atoi(donaldPort)
	attributes := attribute.NewSet(
		semconv.HTTPSchemeHTTP,
		semconv.HTTPFlavorKey.String("1.1"),
		semconv.HTTPMethod("DELETE"),
		semconv.NetPeerName(donaldEndpoint),
		semconv.NetPeerPort(httpserverPortAsInt),
		semconv.HTTPStatusCode(statusCode),
	)

	httpClientDuration.Record(ctx, elapsedTime, attributes.ToSlice()...)
}
