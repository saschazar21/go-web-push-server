package webpush_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"

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

var c *postgres.PostgresContainer

func CreateContainer(ctx context.Context, t *testing.T) (container *postgres.PostgresContainer, err error) {
	if c != nil {
		return c, err
	}

	cwd := os.Getenv("CWD")

	if cwd == "" {
		cwd = CWD
	}

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

	t.Setenv("POSTGRES_CONNECTION_STRING", dsn)

	container = c

	return
}

func TerminateContainer(ctx context.Context, t *testing.T) {
	if c != nil {
		if err := c.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}
}
