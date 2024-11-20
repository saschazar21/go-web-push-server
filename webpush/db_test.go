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
	c, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	t.Run("should connect to database if connection string env is set", func(t *testing.T) {
		t.Cleanup(func() {
			t.Setenv(POSTGRES_CONNECTION_STRING_ENV, "")
		})

		conn, err := ConnectToDatabase()

		assert.NotNil(t, conn)
		assert.Nil(t, err)

		var res sql.Result

		if res, err = conn.Exec("INSERT INTO subscription (endpoint, expiration_time, client_id, recipient_id) VALUES ('https://push.example.com', '2024-11-20', '123', '123') RETURNING recipient_id, client_id"); err != nil {
			t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
		}

		assert.NotNil(t, res)

		type testRecipient struct {
			bun.BaseModel `bun:"table:subscription"`

			ClientId    string `bun:"client_id,pk"`
			RecipientId string `bun:"recipient_id,pk"`
		}

		var r testRecipient

		if err = conn.NewSelect().Model(&testRecipient{}).Scan(ctx, &r); err != nil {
			t.Errorf("TestPostgres err = %v, wantErr = %v", err, nil)
		}

		assert.Equal(t, "123", r.RecipientId)
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
