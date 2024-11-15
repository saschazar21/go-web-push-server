package webpush

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func ConnectToDatabase() (db *bun.DB, err error) {
	dsn := os.Getenv(POSTGRES_CONNECTION_STRING_ENV)

	if dsn == "" {
		return nil, fmt.Errorf("%v env not set", POSTGRES_CONNECTION_STRING_ENV)
	}

	sqlDb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db = bun.NewDB(sqlDb, pgdialect.New())

	return
}
