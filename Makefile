.PHONY: test clean

netlify:
	mkdir -p functions
	go get ./...
	go install ./...

test:
	go test ./...

clean:
	rm -f functions/*