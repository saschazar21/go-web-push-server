.PHONY: test clean

build:
	sh -c ./build.sh

key:
	go run cli/main.go

test:
	go test -coverprofile coverage.out ./...
	
coverage: test
	go tool cover -html coverage.out

clean:
	for d in $(shell ls functions); do rm -rf functions/$$d; done