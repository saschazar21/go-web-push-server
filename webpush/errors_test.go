package webpush

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type customStringerValidator struct {
	Value string `json:"value" validate:"required"`
}

func (c *customStringerValidator) String() string {
	return c.Value
}

func (c *customStringerValidator) Validate() error {
	return CustomValidateStruct(c)
}

func TestErrorResponse(t *testing.T) {
	type test struct {
		name    string
		status  int
		title   string
		other   []string
		cmp     string
		wantErr bool
	}

	tests := []test{
		{
			"validates",
			http.StatusBadRequest,
			"Bad Request",
			[]string{},
			fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", http.StatusBadRequest, "Bad Request"),
			false,
		},
		{
			"fails to validate missing title",
			http.StatusBadRequest,
			"",
			[]string{},
			fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", http.StatusBadRequest, ""),
			true,
		},
		{
			"fails to validate missing status",
			0,
			"Missing",
			[]string{},
			fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", 0, "Missing"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := NewErrorResponse(tt.status, tt.title, tt.other...)

			if err := obj.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("TestErrorResponse err = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.cmp, obj.String())
		})
	}
}

func TestResponseError(t *testing.T) {
	type test struct {
		name    string
		payload StringerValidator
		status  int
		headers []http.Header
		cmp     ResponseError
	}

	tests := []test{
		{
			"creates ErrorResponse",
			&ErrorResponse{
				Errors: []ErrorObject{
					{
						Status: http.StatusMethodNotAllowed,
						Title:  "Method Not Allowed",
					},
				},
			},
			http.StatusMethodNotAllowed,
			[]http.Header{
				{
					http.CanonicalHeaderKey("allow"):        {http.MethodPost},
					http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
				},
			},
			ResponseError{
				fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", http.StatusMethodNotAllowed, "Method Not Allowed"),
				map[string][]string{
					http.CanonicalHeaderKey("allow"):        {http.MethodPost},
					http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
				},
				http.StatusMethodNotAllowed,
			},
		},
		{
			"creates custom StringerValidator",
			&customStringerValidator{
				"Bad Request",
			},
			http.StatusBadRequest,
			[]http.Header{
				{http.CanonicalHeaderKey("content-type"): {TEXT_PLAIN}},
			},
			ResponseError{
				fmt.Sprintf("Bad Request"),
				http.Header{
					http.CanonicalHeaderKey("content-type"): {TEXT_PLAIN},
				},
				http.StatusBadRequest,
			},
		},
		{
			"fails to validate ErrorResponse",
			&ErrorResponse{
				Errors: []ErrorObject{
					{
						Status: http.StatusMethodNotAllowed,
						Title:  "",
					},
				},
			},
			http.StatusMethodNotAllowed,
			[]http.Header{
				{
					http.CanonicalHeaderKey("allow"): {http.MethodPost},
				},
			},
			ResponseError{
				fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", http.StatusInternalServerError, "Internal Server Error"),
				map[string][]string{
					http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
				},
				http.StatusInternalServerError,
			},
		},
		{
			"fails to validate custom StringerValidator",
			&customStringerValidator{
				"",
			},
			http.StatusInternalServerError,
			nil,
			ResponseError{
				fmt.Sprintf("{\"errors\":[{\"status\":%d,\"title\":\"%s\"}]}", http.StatusInternalServerError, "Internal Server Error"),
				map[string][]string{
					http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
				},
				http.StatusInternalServerError,
			},
		},
		{
			"fails to validate custom StringerValidator",
			&customStringerValidator{
				"Internal Server Error",
			},
			0,
			[]http.Header{
				{http.CanonicalHeaderKey("content-type"): {TEXT_PLAIN}},
			},
			ResponseError{
				fmt.Sprintf("Internal Server Error"),
				http.Header{
					http.CanonicalHeaderKey("content-type"): {TEXT_PLAIN},
				},
				http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewResponseError(tt.payload, tt.status, tt.headers...)

			assert.Equal(t, err.Error(), fmt.Sprintf("[HTTP %d]: %s", tt.cmp.StatusCode, tt.cmp.Body))
			assert.Equal(t, tt.cmp, err)

			w := httptest.NewRecorder()

			responseErr, _ := err.(ResponseError)

			responseErr.Response(w)

			assert.Equal(t, tt.cmp.StatusCode, w.Code)
			assert.Equal(t, []byte(tt.cmp.Body), (*w.Body).Bytes())
		})
	}
}
