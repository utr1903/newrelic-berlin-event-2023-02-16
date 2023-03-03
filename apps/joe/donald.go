package main

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
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
	user string,
	reqParams map[string]string,
) error {

	// Get context
	ctx := context.Background()

	log(logrus.InfoLevel, ctx, user, "Preparing HTTP call...")

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
		log(logrus.ErrorLevel, ctx, user, err.Error())
		return err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-User-ID", user)

	// Add request params
	qps := req.URL.Query()
	for k, v := range reqParams {
		qps.Add(k, v)
	}
	if len(qps) > 0 {
		req.URL.RawQuery = qps.Encode()
		log(logrus.InfoLevel, ctx, user, "Request params->"+req.URL.RawQuery)
	}
	log(logrus.InfoLevel, ctx, user, "HTTP call is prepared.")

	// Start timer
	requestStartTime := time.Now()

	// Perform HTTP request
	log(logrus.InfoLevel, ctx, user, "Performing HTTP call")
	res, err := httpClient.Do(req)
	if err != nil {
		log(logrus.ErrorLevel, ctx, user, err.Error())
		recordClientDuration(ctx, httpMethod, http.StatusInternalServerError, requestStartTime)
		return err
	}
	defer res.Body.Close()

	// Read HTTP response
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log(logrus.ErrorLevel, ctx, user, err.Error())
		recordClientDuration(ctx, httpMethod, res.StatusCode, requestStartTime)
		return err
	}

	// Check status code
	if res.StatusCode != http.StatusOK {
		log(logrus.ErrorLevel, ctx, user, string(resBody))
		recordClientDuration(ctx, httpMethod, res.StatusCode, requestStartTime)
		return errors.New("call to donald returned not ok status")
	}

	recordClientDuration(ctx, httpMethod, res.StatusCode, requestStartTime)
	log(logrus.InfoLevel, ctx, user, "HTTP call is performed successfully.")
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
