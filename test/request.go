package webpush_test

import (
	"net/http"
	"strings"
)

func EchoHeaders(res http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		res.Header().Add(k, strings.Join(v, ", "))
	}

	res.WriteHeader(200)
	res.Write([]byte("OK"))
}
