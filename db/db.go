package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/saschazar21/go-web-push-server/utils"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func Connect() (db *bun.DB, err error) {
	dsn := os.Getenv(utils.POSTGRES_CONNECTION_STRING_ENV)

	if dsn == "" {
		return nil, fmt.Errorf("%v env not set", utils.POSTGRES_CONNECTION_STRING_ENV)
	}

	sqlDb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db = bun.NewDB(sqlDb, pgdialect.New())

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv("DEBUG"),
	))

	return
}
