.PHONY: test clean

build:
	sh -c ./build.sh
	npx eleventy
	npx workbox injectManifest workbox-config.cjs

key:
	go run cli/main.go

test:
	go test -coverprofile coverage.out ./...
	
coverage: test
	go tool cover -html coverage.out

clean:
	rm -rf public
	for d in $(shell ls functions); do rm -rf functions/$$d; done