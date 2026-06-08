package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() error {
	dbDriver := os.Getenv("DB_DRIVER")
	dbUrl := os.Getenv("DB_URL")

	var err error

	DB, err = sql.Open(dbDriver, dbUrl)
	if err != nil {
		panic(err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	return DB.Ping()
}
