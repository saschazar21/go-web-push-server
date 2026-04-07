.PHONY: test clean

build: build_website
	sh -c ./build.sh

build_website:
	mkdir -p public
ifdef ENABLE_DEMO
	npx eleventy
	npx workbox injectManifest workbox-config.cjs
endif

key:
	@go run cli/main.go

test:
	@echo "Running tests..."
	@go test ./... -v -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Tests completed."

clean:
	rm -rf public
	for d in $(shell ls functions); do rm -rf functions/$$d; done