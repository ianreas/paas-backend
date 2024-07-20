// db/connection.go
package db

import (
    "database/sql"
    _ "github.com/lib/pq" // Postgres driver
    "log"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB(dataSourceName string) {
    var err error
    DB, err = sql.Open("postgres", dataSourceName)
    if err != nil {
        log.Fatalf("Error opening database: %q", err)
    }
}
