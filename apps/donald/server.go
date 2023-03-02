package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func listHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodGet {
		createHttpResponse(&w, http.StatusMethodNotAllowed, []byte("Method not allowed"))
	}

	// Build db query
	dbOperation := "SELECT"
	dbStatement := dbOperation + " name FROM " + mysqlTable

	// Get current parentSpan
	parentSpan := trace.SpanFromContext(r.Context())
	defer parentSpan.End()

	// Create db span
	if considerDatabaseSpans {
		_, dbSpan := parentSpan.TracerProvider().
			Tracer(appName).
			Start(
				r.Context(),
				dbOperation+" "+mysqlDatabase+"."+mysqlTable,
				trace.WithSpanKind(trace.SpanKindClient),
			)
		defer dbSpan.End()

		// Set additional span attributes
		dbSpan.SetAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.user", mysqlUsername),
			attribute.String("net.peer.name", mysqlServer),
			attribute.String("net.peer.port", mysqlPort),
			attribute.String("net.transport", "IP.TCP"),
			attribute.String("db.name", mysqlDatabase),
			attribute.String("db.sql.table", mysqlTable),
			attribute.String("db.statement", dbStatement),
			attribute.String("db.operation", dbOperation),
		)
	}

	// Perform a query
	rows, err := db.Query(dbStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Iterate over the results
	names := make([]string, 0, 10)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			fmt.Println(err)
			createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()))
			break
		}
		names = append(names, name)
	}

	resBody, err := json.Marshal(names)
	if err != nil {
		fmt.Println(err)
		createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()))
	}

	createHttpResponse(&w, http.StatusOK, resBody)
}

func deleteHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	// Get current parentSpan
	parentSpan := trace.SpanFromContext(r.Context())
	defer parentSpan.End()

	if r.Method != http.MethodDelete {
		createHttpResponse(&w, http.StatusMethodNotAllowed, []byte("Method not allowed"))
		return
	}

	// Parse query parameters
	createDatabaseConnectionError := r.URL.Query().Get("createDatabaseConnectionError")

	// Build query
	dbOperation := "DELETE"
	dbStatement := dbOperation + " FROM " + mysqlTable

	// Create db span
	if considerDatabaseSpans {
		_, dbSpan := parentSpan.TracerProvider().
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
		_, err := db.Exec(dbStatement)
		if err != nil {
			fmt.Println(err.Error())

			// Add status code
			dbSpanAttrs = append(dbSpanAttrs, attribute.String("otel.status_code", "ERROR"))
			dbSpan.SetAttributes(dbSpanAttrs...)

			createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()))
			return
		}

		if createDatabaseConnectionError == "true" {
			fmt.Println("Connection to database is lost.")

			// Add status code
			dbSpanAttrs = append(dbSpanAttrs, attribute.String("otel.status_code", "ERROR"))
			dbSpan.SetAttributes(dbSpanAttrs...)

			createHttpResponse(&w, http.StatusInternalServerError, []byte("Connection to database is lost."))
			return
		}
		dbSpan.SetAttributes(dbSpanAttrs...)
	} else {

		// Perform query
		_, err := db.Exec(dbStatement)
		if err != nil {
			fmt.Println(err.Error())
			createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()))
			return
		}

		if createDatabaseConnectionError == "true" {
			fmt.Println("Connection to database is lost.")
			createHttpResponse(&w, http.StatusInternalServerError, []byte("Connection to database is lost."))
			return
		}
	}

	createHttpResponse(&w, http.StatusOK, []byte("Success"))
}

func createHttpResponse(
	w *http.ResponseWriter,
	statusCode int,
	body []byte,
) {
	(*w).WriteHeader(statusCode)
	(*w).Write(body)
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
