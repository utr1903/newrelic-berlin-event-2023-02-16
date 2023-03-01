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

	if r.Method != http.MethodDelete {
		createHttpResponse(&w, http.StatusMethodNotAllowed, []byte("Method not allowed"))
	}

	// Build query
	dbOperation := "DELETE"
	dbStatement := dbOperation + " FROM " + mysqlTable

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

	// Perform query
	_, err := db.Exec(dbStatement)
	if err != nil {
		fmt.Println(err.Error())
		createHttpResponse(&w, http.StatusInternalServerError, []byte(err.Error()))
		return
	}

	createHttpResponse(&w, http.StatusOK, []byte("Success"))
}

func createHttpResponse(
	w *http.ResponseWriter,
	statusCode uint,
	body []byte,
) {
	(*w).WriteHeader(http.StatusOK)
	(*w).Write(body)
}
