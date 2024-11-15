package webpush

import (
	"context"
	"database/sql"
	"testing"

	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
)

func TestConnectToDatabase(t *testing.T) {
	ctx := context.Background()

	// create test container
	_, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	defer webpush_test.TerminateContainer(ctx, t)

	t.Run("should connect to database if connection string env is set", func(t *testing.T) {
		t.Cleanup(func() {
			t.Setenv(POSTGRES_CONNECTION_STRING_ENV, "")
		})

		conn, err := ConnectToDatabase()

		assert.NotNil(t, conn)
		assert.Nil(t, err)

		var res sql.Result

		if res, err = conn.Exec("INSERT INTO recipient (id, client_id) VALUES ('123', '123') RETURNING id, client_id"); err != nil {
			t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
		}

		assert.NotNil(t, res)

		type testRecipient struct {
			bun.BaseModel `bun:"table:recipient"`

			ID       string `bun:"id,pk"`
			ClientId string `bun:"client_id,pk"`
		}

		var r testRecipient

		if err = conn.NewSelect().Model(&testRecipient{}).Scan(ctx, &r); err != nil {
			t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
		}

		assert.Equal(t, "123", r.ID)
		assert.Equal(t, "123", r.ClientId)
	})

	t.Run("should return error if connection string env is not set", func(t *testing.T) {
		t.Cleanup(func() {
			t.Setenv(POSTGRES_CONNECTION_STRING_ENV, "")
		})

		// Unset the environment variable
		t.Setenv(POSTGRES_CONNECTION_STRING_ENV, "")

		conn, err := ConnectToDatabase()

		assert.Nil(t, conn)
		assert.NotNil(t, err)
		assert.Equal(t, "POSTGRES_CONNECTION_STRING env not set", err.Error())
	})
}
