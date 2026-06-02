package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() error {
	connStr := "postgres://postgres:123@localhost/vms?sslmode=disable"
	var err error

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	return DB.Ping()
}
