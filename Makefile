.PHONY: test clean

build: build_website
	mkdir -p public
	sh -c ./build.sh

build_website:
ifdef ENABLE_DEMO
	npx eleventy
	npx workbox injectManifest workbox-config.cjs
endif

key:
	go run cli/main.go

test:
	go test -coverprofile coverage.out ./...
	
coverage: test
	go tool cover -html coverage.out

clean:
	rm -rf public
	for d in $(shell ls functions); do rm -rf functions/$$d; done