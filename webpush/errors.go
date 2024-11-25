package webpush

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var (
	BAD_REQUEST_ERROR = &ErrorResponse{
		[]ErrorObject{
			{
				Status: http.StatusBadRequest,
				Title:  "Bad Request",
			},
		},
	}
	UNAUTHORIZED_ERROR = &ErrorResponse{
		[]ErrorObject{
			{
				Status: http.StatusUnauthorized,
				Title:  "Unauthorized",
			},
		},
	}
	FORBIDDEN_ERROR = &ErrorResponse{
		[]ErrorObject{
			{
				Status: http.StatusForbidden,
				Title:  "Forbidden",
			},
		},
	}
	METHOD_NOT_ALLOWED_ERROR = &ErrorResponse{
		[]ErrorObject{
			{
				Status: http.StatusMethodNotAllowed,
				Title:  "Method Not Allowed",
			},
		},
	}
	INTERNAL_SERVER_ERROR = &ErrorResponse{
		[]ErrorObject{
			{
				Status: http.StatusInternalServerError,
				Title:  "Internal Server Error",
			},
		},
	}
)

type ErrorMeta struct {
	Endpoint string `json:"endpoint,omitempty"`
}

type ErrorObject struct {
	Status int        `json:"status" validate:"required"`
	Code   string     `json:"code,omitempty"`
	Title  string     `json:"title" validate:"required"`
	Detail string     `json:"detail,omitempty"`
	Meta   *ErrorMeta `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Errors []ErrorObject `json:"errors" validate:"required,dive"`
}

func (res *ErrorResponse) Add(err ErrorObject) {
	res.Errors = append(res.Errors, err)
}

func (res *ErrorResponse) String() string {
	buf, _ := json.Marshal(res)

	return string(buf)
}

func (res *ErrorResponse) Validate() (err error) {
	if err = CustomValidateStruct(res); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError, http.Header{http.CanonicalHeaderKey("content-type"): {JSON_API}})
	}

	return
}

func NewErrorResponse(status int, title string, other ...string) *ErrorResponse {
	secondary := make([]string, 2)
	copy(other, secondary)

	return &ErrorResponse{
		[]ErrorObject{
			{
				status,
				secondary[1],
				title,
				secondary[0],
				nil,
			},
		},
	}
}

type ResponseError struct {
	Body       string
	Headers    http.Header
	StatusCode int
}

func (err ResponseError) Error() (s string) {
	return fmt.Sprintf("[HTTP %d]: %s", err.StatusCode, err.Body)
}

func (err ResponseError) Write(w http.ResponseWriter) {
	if err.Headers != nil {
		for key, value := range err.Headers {
			w.Header().Add(key, strings.Join(value, ","))
		}
	}

	w.WriteHeader(err.StatusCode)

	w.Write([]byte(err.Body))
}

func NewResponseError(contents StringerValidator, status int, headers ...http.Header) (err error) {
	if len(headers) < 1 {
		headers = []http.Header{
			{http.CanonicalHeaderKey("content-type"): {JSON_API}},
		}
	}

	if err = contents.Validate(); err != nil {
		responseErr, ok := err.(ResponseError)

		if ok {
			return responseErr
		}

		log.Println(err)

		contents = INTERNAL_SERVER_ERROR
		status = http.StatusInternalServerError
		headers = []http.Header{
			{http.CanonicalHeaderKey("content-type"): {JSON_API}},
		}
	}

	if status < http.StatusOK {
		status = http.StatusInternalServerError
	}

	return ResponseError{
		contents.String(),
		headers[0],
		status,
	}
}

func WriteResponseError(w http.ResponseWriter, err error, fallback ...error) {
	responseErr, ok := err.(ResponseError)

	if !ok {
		if len(fallback) > 0 {
			responseErr, ok = fallback[0].(ResponseError)

			if !ok {
				responseErr = NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError).(ResponseError)
			}

		} else {
			log.Println(err)

			responseErr = NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError).(ResponseError)
		}
	}

	responseErr.Write(w)
}
