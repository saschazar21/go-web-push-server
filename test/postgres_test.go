package webpush_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgres(t *testing.T) {
	var c *postgres.PostgresContainer
	var err error

	ctx := context.Background()

	if c, err = CreateContainer(ctx, t); err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	var result int
	var reader io.Reader

	if result, reader, err = c.Exec(ctx, []string{"psql", "-U", DB_USER, "-d", DB_PASSWORD, "-c", "SELECT 1"}); err != nil {
		t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	assert.Equal(t, 0, result)

	buf := make([]byte, 1)

	if _, err = reader.Read(buf); err != nil {
		t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	assert.Equal(t, uint8(1), buf[0])
}
