package api_utils

import (
	"log"
	"net/http"
	"regexp"

	"github.com/saschazar21/go-web-push-server/webpush"
)

func HandleURLRegex(r *http.Request, pattern string) (values []string, names []string, err error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		log.Println(err)
		log.Printf("invalid pattern %s\n", pattern)

		err = webpush.NewResponseError(webpush.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
		return
	}

	matches := regex.FindStringSubmatch(r.URL.Path)
	if len(matches) == 0 {
		return
	}

	names = regex.SubexpNames()[1:]
	values = matches[1:]

	return
}
