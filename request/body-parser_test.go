package request

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saschazar21/go-web-push-server/utils"
)

func TestParseBody(t *testing.T) {

	type testCase struct {
		name    string
		method  string
		body    []byte
		wantErr bool
	}

	tests := []testCase{
		{
			name:    "should parse valid JSON body",
			method:  http.MethodPost,
			body:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "should return error on invalid JSON body",
			method:  http.MethodPost,
			body:    []byte(`{"key":value}`),
			wantErr: true,
		},
		{
			name:    "should return error on non-POST method",
			method:  http.MethodGet,
			body:    []byte(`{"key":"value"}`),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "https:///api/v1/test", bufio.NewReader(bytes.NewReader(tc.body)))
			req.Header.Set("content-type", utils.APPLICATION_JSON)

			var result map[string]string
			err := ParseBody(req, &result)

			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error: %v, got: %v", tc.wantErr, err)
			}

			if !tc.wantErr {
				if val, ok := result["key"]; !ok || val != "value" {
					t.Errorf("expected key to be 'value', got '%s'", val)
				}
			}
		})
	}
}
