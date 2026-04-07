package webpush_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/saschazar21/go-web-push-server/utils"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	IMAGE_NAME = "postgres:alpine"

	DB_NAME     = "test"
	DB_USER     = "postgres"
	DB_PASSWORD = "postgres"

	DB_LOG    = "database system is ready to accept connections"
	DB_SCHEMA = "schema.sql"

	CWD = "../"
)

func CreateContainer(ctx context.Context, t *testing.T) (container *postgres.PostgresContainer, err error) {
	cwd := os.Getenv("CWD")

	if cwd == "" {
		cwd = CWD
	}

	var c *postgres.PostgresContainer

	if c, err = postgres.Run(
		ctx,
		IMAGE_NAME,
		postgres.WithDatabase(DB_NAME),
		postgres.WithUsername(DB_USER),
		postgres.WithPassword(DB_PASSWORD),
		postgres.WithInitScripts(fmt.Sprintf("%v%v", cwd, DB_SCHEMA)),
		postgres.BasicWaitStrategies(),
	); err != nil {
		return
	}

	if err = c.Snapshot(ctx, postgres.WithSnapshotName("test-postgres-initial")); err != nil {
		log.Fatalf("failed to create snapshot: %v", err)
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")

	if err != nil {
		log.Fatalf("failed to retrieve DB connection string: %v", err)
	}

	t.Setenv(utils.MASTER_KEY_ENV, "l342tf9eC2l4/fVytEkkzQzYyqd3eKd6GViw65WB5yI=")
	t.Setenv(utils.POSTGRES_CONNECTION_STRING_ENV, dsn)
	t.Setenv("DEBUG", "2")

	container = c

	return
}
