package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

	// Perform database query
	if considerDatabaseSpans {
		err := performQueryWithDbSpan(w, r, &parentSpan)
		if err != nil {
			return
		}
	} else {
		err := performQueryWithoutDbSpan(w, r, &parentSpan)
		if err != nil {
			return
		}
	}

	createHttpResponse(&w, http.StatusOK, []byte("Success"), &parentSpan)
}

func performQueryWithDbSpan(
	w http.ResponseWriter,
	r *http.Request,
	parentSpan *trace.Span,
) error {

	// Build query
	dbOperation, dbStatement, err := createDbQuery(r.Method)
	if err != nil {
		createHttpResponse(&w, http.StatusMethodNotAllowed, []byte("Method not allowed"), parentSpan)
		return err
	}

	_, dbSpan := (*parentSpan).TracerProvider().
		Tracer(appName).
		Start(
			r.Context(),
			dbOperation+" "+mysqlDatabase+"."+mysqlTable,
			trace.WithSpanKind(trace.SpanKindClient),
		)
	defer dbSpan.End()

	// Set additional span attributes
	dbSpanAttrs := getCommonDbSpanAttributes()
	dbSpanAttrs = append(dbSpanAttrs, attribute.String("db.statement", dbStatement))
	dbSpanAttrs = append(dbSpanAttrs, attribute.String("db.operation", dbOperation))

	// Perform query
	err = executeDbQuery(r.Method, dbStatement)
	if err != nil {
		// Add status code
		dbSpanAttrs = append(dbSpanAttrs, attribute.String("otel.status_code", "ERROR"))
		dbSpan.SetAttributes(dbSpanAttrs...)

		createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()), parentSpan)
		return err
	}

	// Parse query parameters
	createDatabaseConnectionError := r.URL.Query().Get("createDatabaseConnectionError")
	if createDatabaseConnectionError == "true" {
		fmt.Println("Connection to database is lost.")

		// Add status code
		dbSpanAttrs = append(dbSpanAttrs, attribute.String("otel.status_code", "ERROR"))
		dbSpan.SetAttributes(dbSpanAttrs...)

		createHttpResponse(&w, http.StatusInternalServerError, []byte("Connection to database is lost."), parentSpan)
		return errors.New("database connection lost")
	}
	dbSpan.SetAttributes(dbSpanAttrs...)
	return nil
}

func performQueryWithoutDbSpan(
	w http.ResponseWriter,
	r *http.Request,
	parentSpan *trace.Span,
) error {
	// Build query
	_, dbStatement, err := createDbQuery(r.Method)
	if err != nil {
		createHttpResponse(&w, http.StatusMethodNotAllowed, []byte("Method not allowed"), parentSpan)
		return err
	}

	// Perform query
	err = executeDbQuery(r.Method, dbStatement)
	if err != nil {
		createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()), parentSpan)
		return err
	}

	// Parse query parameters
	createDatabaseConnectionError := r.URL.Query().Get("createDatabaseConnectionError")
	if createDatabaseConnectionError == "true" {
		fmt.Println("Connection to database is lost.")
		createHttpResponse(&w, http.StatusInternalServerError, []byte("Connection to database is lost."), parentSpan)
		return errors.New("database connection lost")
	}
	return nil
}

func createDbQuery(
	httpMethod string,
) (
	string,
	string,
	error,
) {
	switch httpMethod {
	case http.MethodGet:
		dbOperation := "SELECT"
		dbStatement := dbOperation + " name FROM " + mysqlTable
		return dbOperation, dbStatement, nil
	case http.MethodDelete:
		dbOperation := "DELETE"
		dbStatement := dbOperation + " FROM " + mysqlTable
		return dbOperation, dbStatement, nil
	default:
		return "", "", errors.New("method not allowed")
	}
}

func executeDbQuery(
	httpMethod string,
	dbStatement string,
) error {

	switch httpMethod {
	case http.MethodGet:
		// Perform a query
		rows, err := db.Query(dbStatement)
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer rows.Close()

		// Iterate over the results
		names := make([]string, 0, 10)
		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			if err != nil {
				fmt.Println(err)
				return err
			}
			names = append(names, name)
		}

		_, err = json.Marshal(names)
		if err != nil {
			fmt.Println(err)
			return err
		}
	case http.MethodDelete:
		_, err := db.Exec(dbStatement)
		if err != nil {
			fmt.Println(err)
			return err
		}
	default:
		return errors.New("method not allowed")
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

func getCommonDbSpanAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("db.system", "mysql"),
		attribute.String("db.user", mysqlUsername),
		attribute.String("net.peer.name", mysqlServer),
		attribute.String("net.peer.port", mysqlPort),
		attribute.String("net.transport", "IP.TCP"),
		attribute.String("db.name", mysqlDatabase),
		attribute.String("db.sql.table", mysqlTable),
	}
}
