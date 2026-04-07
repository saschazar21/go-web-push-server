package webpush_test

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadFixture(name string, v interface{}) (err error) {
	contents, err := os.ReadFile(filepath.Join("../test/fixtures/", name))
	if err != nil {
		return err
	}
	return json.Unmarshal(contents, v)
}
