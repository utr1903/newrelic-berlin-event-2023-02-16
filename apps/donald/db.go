package main

import (
	"database/sql"
	"fmt"
	"os"
)

var (
	mysqlServer   = os.Getenv("MYSQL_SERVER")
	mysqlUsername = os.Getenv("MYSQL_USERNAME")
	mysqlPassword = os.Getenv("MYSQL_PASSWORD")
	mysqlDatabase = os.Getenv("MYSQL_DATABASE")
	mysqlTable    = os.Getenv("MYSQL_TABLE")
	mysqlPort     = os.Getenv("MYSQL_PORT")

	db *sql.DB
)

func createDatabaseConnection() *sql.DB {
	// Connect to MySQL
	datasourceName := mysqlUsername + ":" + mysqlPassword + "@tcp(" + mysqlServer + ":" + mysqlPort + ")/"
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		panic(err)
	}

	// Create the database
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + mysqlDatabase)
	if err != nil {
		panic(err)
	}

	fmt.Println("Database is created successfully!")

	// Use the database
	_, err = db.Exec("USE " + mysqlDatabase)
	if err != nil {
		panic(err)
	}

	// Create the table
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + mysqlTable + " (id INT NOT NULL PRIMARY KEY AUTO_INCREMENT, name VARCHAR(50) NOT NULL)")
	if err != nil {
		panic(err)
	}

	fmt.Println("Table is created successfully!")
	return db
}
