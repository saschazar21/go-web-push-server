package webpush

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCustomValidator(t *testing.T) {
	type test struct {
		name    string
		value   any
		wantErr bool
	}

	tests := []test{
		{
			"valid epoch-gt-now",
			struct {
				Val int64 `validate:"epoch-gt-now"`
			}{
				time.Now().Add(3 * time.Second).Unix(),
			},
			false,
		},
		{
			"invalid epoch-gt-now",
			struct {
				Val int64 `validate:"epoch-gt-now"`
			}{
				time.Now().Add(-3 * time.Second).Unix(),
			},
			true,
		},
		{
			"valid origin",
			struct {
				Val string `validate:"origin"`
			}{
				"https://github.com:8080",
			},
			false,
		},
		{
			"invalid origin",
			struct {
				Val string `validate:"origin"`
			}{
				"https://github.com:8080/trapped/forever",
			},
			true,
		},
		{
			"invalid type origin",
			struct {
				Val float32 `validate:"origin"`
			}{
				1.3,
			},
			true,
		},
		{
			"valid mailto",
			struct {
				Val string `validate:"mailto"`
			}{
				"mailto:test@example.com",
			},
			false,
		},
		{
			"invalid mailto",
			struct {
				Val string `validate:"mailto"`
			}{
				"mailto:somelink.com",
			},
			true,
		},
		{
			"invalid type mailto",
			struct {
				Val float32 `validate:"mailto"`
			}{
				1.3,
			},
			true,
		},
		{
			"invalid url",
			struct {
				Val string `validate:"url"`
			}{
				"asdf",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := NewCustomValidator()

			assert.NotNil(t, validate)

			if err := CustomValidateStruct(tt.value); (err != nil) != tt.wantErr {
				t.Errorf("TestCustomValidator err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
